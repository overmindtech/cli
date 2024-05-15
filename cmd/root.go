package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

var logLevel string

//go:generate sh -c "echo -n $(git describe --tags --long) > commit.txt"
//go:embed commit.txt
var cliVersion string

type OvermindInstance struct {
	FrontendUrl *url.URL
	ApiUrl      *url.URL
	NatsUrl     *url.URL
	Audience    string
}

// GatewayUrl returns the URL for the gateway for this instance.
func (oi OvermindInstance) GatewayUrl() string {
	return fmt.Sprintf("%v/api/gateway", oi.ApiUrl.String())
}

func (oi OvermindInstance) String() string {
	return fmt.Sprintf("Frontend: %v, API: %v, Nats: %v, Audience: %v", oi.FrontendUrl, oi.ApiUrl, oi.NatsUrl, oi.Audience)
}

type instanceData struct {
	Api  string `json:"api_url"`
	Nats string `json:"nats_url"`
	Aud  string `json:"aud"`
}

// NewOvermindInstance creates a new OvermindInstance from the given app URL
// with all URLs filled in, or an error. This makes a request to the frontend to
// lookup Api and Nats URLs.
func NewOvermindInstance(ctx context.Context, app string) (OvermindInstance, error) {
	var instance OvermindInstance
	var err error

	instance.FrontendUrl, err = url.Parse(app)
	if err != nil {
		return instance, fmt.Errorf("invalid --app value '%v', error: %w", app, err)
	}

	// Get the instance data
	instanceDataUrl := fmt.Sprintf("%v/api/public/instance-data", instance.FrontendUrl)
	req, err := http.NewRequest("GET", instanceDataUrl, nil)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("could not initialize instance-data fetch")
		return OvermindInstance{}, fmt.Errorf("could not initialize instance-data fetch: %w", err)
	}

	req = req.WithContext(ctx)
	log.WithField("instanceDataUrl", instanceDataUrl).Debug("Fetching instance-data")
	res, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("could not fetch instance-data: %w", err)
	}

	if res.StatusCode != 200 {
		return OvermindInstance{}, fmt.Errorf("instance-data fetch returned non-200 status: %v", res.StatusCode)
	}

	defer res.Body.Close()
	data := instanceData{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("could not parse instance-data: %w", err)
	}

	instance.ApiUrl, err = url.Parse(data.Api)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("invalid api_url value '%v' in instance-data, error: %w", data.Api, err)
	}
	instance.NatsUrl, err = url.Parse(data.Nats)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("invalid nats_url value '%v' in instance-data, error: %w", data.Nats, err)
	}

	instance.Audience = data.Aud

	return instance, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "overmind",
	Short: "The Overmind CLI",
	Long: `Calculate the blast radius of your changes, track risks, and make changes with
confidence.

This CLI will prompt you for authentication using Overmind's OAuth service,
however it can also be configured to use an API key by setting the OVM_API_KEY
environment variable.`,
	Version: cliVersion,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `root` flags")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type statusMsg int

const (
	PromptUser             statusMsg = 0
	WaitingForConfirmation statusMsg = 1
	Authenticated          statusMsg = 2
	ErrorAuthenticating    statusMsg = 3
)

type authenticateModel struct {
	ctx context.Context

	status     statusMsg
	err        error
	deviceCode *oauth2.DeviceAuthResponse
	config     oauth2.Config
	token      *oauth2.Token
}

func (m authenticateModel) Init() tea.Cmd {
	return openBrowserCmd(m.deviceCode.VerificationURI)
}

func (m authenticateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		default:
			{
				if m.status == Authenticated {
					return m, tea.Quit
				}
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case *oauth2.Token:
		{
			m.status = Authenticated
			m.token = msg
			return m, nil
		}

	case statusMsg:
		switch msg {
		case PromptUser:
			return m, openBrowserCmd(m.deviceCode.VerificationURI)
		case WaitingForConfirmation:
			m.status = WaitingForConfirmation
			return m, awaitToken(m.ctx, m.config, m.deviceCode)
		case Authenticated:
		case ErrorAuthenticating:
			{
				return m, nil
			}
		}

	case displayAuthorizationInstructionsMsg:
		m.status = WaitingForConfirmation
		return m, awaitToken(m.ctx, m.config, m.deviceCode)

	case failedToAuthenticateErrorMsg:
		m.err = msg.err
		m.status = ErrorAuthenticating
		return m, tea.Quit

	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func (m authenticateModel) View() string {
	var output string
	beginAuthMessage := `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

%v

Then enter the code:

	%v
`
	prompt := fmt.Sprintf(beginAuthMessage, m.deviceCode.VerificationURI, m.deviceCode.UserCode)
	output += markdownToString(prompt)
	switch m.status {
	case PromptUser:
		// nothing here as PromptUser is the default
	case WaitingForConfirmation:
		// sp := createSpinner()
		// output += sp.View() + " Waiting for confirmation..."
	case Authenticated:
		output = "✅ Authenticated successfully. Press any key to continue."
	case ErrorAuthenticating:
		output = "⛔️ Unable to authenticate. Try again."
	}

	return containerStyle.Render(output)
}

type errMsg struct{ err error }
type failedToAuthenticateErrorMsg struct{ err error }

func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		err := browser.OpenURL(url)
		if err != nil {
			return displayAuthorizationInstructionsMsg{deviceCode: nil, err: err}
		}
		return WaitingForConfirmation
	}
}

func awaitToken(ctx context.Context, config oauth2.Config, deviceCode *oauth2.DeviceAuthResponse) tea.Cmd {
	return func() tea.Msg {
		token, err := config.DeviceAccessToken(ctx, deviceCode)
		if err != nil {
			return failedToAuthenticateErrorMsg{err}
		}

		return token
	}
}

// getChangeUuid returns the UUID of a change, as selected by --uuid or --change, or a state with the specified status and having --ticket-link
func getChangeUuid(ctx context.Context, oi OvermindInstance, expectedStatus sdp.ChangeStatus, ticketLink string, errNotFound bool) (uuid.UUID, error) {
	var changeUuid uuid.UUID
	var err error

	uuidString := viper.GetString("uuid")
	changeUrlString := viper.GetString("change")

	// If no arguments are specified then return an error
	if uuidString == "" && changeUrlString == "" && ticketLink == "" {
		return uuid.Nil, errors.New("no change specified; use one of --change, --ticket-link or --uuid")
	}

	// Check UUID first if more than one is set
	if uuidString != "" {
		changeUuid, err = uuid.Parse(uuidString)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid --uuid value '%v', error: %w", uuidString, err)
		}

		return changeUuid, nil
	}

	// Then check for a change URL
	if changeUrlString != "" {
		return parseChangeUrl(changeUrlString)
	}

	// Finally look through all open changes to find one with a matching ticket link
	client := AuthenticatedChangesClient(ctx, oi)

	changesList, err := client.ListChangesByStatus(ctx, &connect.Request[sdp.ListChangesByStatusRequest]{
		Msg: &sdp.ListChangesByStatusRequest{
			Status: expectedStatus,
		},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to search for existing changes: %w", err)
	}

	var maybeChangeUuid *uuid.UUID
	for _, c := range changesList.Msg.GetChanges() {
		if c.GetProperties().GetTicketLink() == ticketLink {
			maybeChangeUuid = c.GetMetadata().GetUUIDParsed()
			if maybeChangeUuid != nil {
				changeUuid = *maybeChangeUuid
				break
			}
		}
	}

	if errNotFound && changeUuid == uuid.Nil {
		return uuid.Nil, fmt.Errorf("no change found with ticket link %v", ticketLink)
	}

	return changeUuid, nil
}

func parseChangeUrl(changeUrlString string) (uuid.UUID, error) {
	changeUrl, err := url.ParseRequestURI(changeUrlString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', error: %w", changeUrlString, err)
	}
	pathParts := strings.Split(path.Clean(changeUrl.Path), "/")
	if len(pathParts) < 2 {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', not long enough: %w", changeUrlString, err)
	}
	changeUuid, err := uuid.Parse(pathParts[2])
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', couldn't parse UUID: %w", changeUrlString, err)
	}
	return changeUuid, nil
}

func addChangeUuidFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("change", "", "The frontend URL of the change to get")
	cmd.PersistentFlags().String("ticket-link", "", "Link to the ticket for this change.")
	cmd.PersistentFlags().String("uuid", "", "The UUID of the change that should be displayed.")
	cmd.MarkFlagsMutuallyExclusive("change", "ticket-link", "uuid")
}

// Adds common flags to API commands e.g. timeout
func addAPIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("timeout", "10m", "How long to wait for responses")
	cmd.PersistentFlags().String("app", "https://app.overmind.tech", "The overmind instance to connect to.")
}

func init() {
	cobra.OnInitialize(initConfig)

	// General Config
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")
	cobra.CheckErr(viper.BindEnv("log", "OVERMIND_LOG", "LOG")) // fallback to global config

	// Support API Keys in the environment
	err := viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY")
	if err != nil {
		log.WithError(err).Fatal("could not bind api key to env")
	}

	// internal configs
	rootCmd.PersistentFlags().String("cli-auth0-client-id", "QMfjMww3x4QTpeXiuRtMV3JIQkx6mZa4", "OAuth Client ID to use when connecting with auth0")
	rootCmd.PersistentFlags().String("cli-auth0-domain", "om-prod.eu.auth0.com", "Auth0 domain to connect to")
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb. This requires --otel to be set.")
	rootCmd.PersistentFlags().String("ovm-test-fake", "", "If non-empty, instructs some commands to only use fake data for fast development iteration.")

	// Mark these as hidden. This means that it will still be parsed of supplied,
	// and we will still look for it in the environment, but it won't be shown
	// in the help
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("cli-auth0-client-id"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("cli-auth0-domain"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("honeycomb-api-key"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("ovm-test-fake"))

	// Create groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "iac",
		Title: "Infrastructure as Code:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "api",
		Title: "Overmind API:",
	})

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		formatter := new(log.TextFormatter)
		formatter.DisableTimestamp = true
		log.SetFormatter(formatter)

		// Read env vars
		var lvl log.Level

		if logLevel != "" {
			lvl, err = log.ParseLevel(logLevel)
			if err != nil {
				log.WithFields(log.Fields{"level": logLevel, "err": err}).Errorf("couldn't parse `log` config, defaulting to `info`")
				lvl = log.InfoLevel
			}
		} else {
			lvl = log.ErrorLevel
		}
		log.SetLevel(lvl)

		if honeycombApiKey := viper.GetString("honeycomb-api-key"); honeycombApiKey != "" {
			if err := tracing.InitTracerWithHoneycomb(honeycombApiKey); err != nil {
				log.Fatal(err)
			}

			log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
				log.AllLevels[:log.GetLevel()+1]...,
			)))

			// shut down tracing at the end of the process
			rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
				tracing.ShutdownTracer()
			}
		}
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	replacer := strings.NewReplacer("-", "_")

	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match
}
