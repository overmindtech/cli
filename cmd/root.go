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
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

//go:generate sh -c "echo -n $(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD) > commit.txt"
//go:embed commit.txt
var cliVersion string

type OvermindInstance struct {
	FrontendUrl *url.URL
	ApiUrl      *url.URL
	NatsUrl     *url.URL
	Audience    string
	Auth0Domain string
	CLIClientID string
}

// GatewayUrl returns the URL for the gateway for this instance.
func (oi OvermindInstance) GatewayUrl() string {
	return fmt.Sprintf("%v/api/gateway", oi.ApiUrl.String())
}

func (oi OvermindInstance) String() string {
	return fmt.Sprintf("Frontend: %v, API: %v, Nats: %v, Audience: %v", oi.FrontendUrl, oi.ApiUrl, oi.NatsUrl, oi.Audience)
}

type instanceData struct {
	Api         string `json:"api_url"`
	Nats        string `json:"nats_url"`
	Aud         string `json:"aud"`
	Auth0Domain string `json:"auth0_domain"`
	CLIClientID string `json:"auth0_cli_client_id"`
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
	instance.CLIClientID = data.CLIClientID
	instance.Auth0Domain = data.Auth0Domain

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
	Version:      cliVersion,
	SilenceUsage: true,
	PreRun:       PreRunSetup,
}

var cmdSpan trace.Span

func PreRunSetup(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	// Bind these to viper
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		log.WithError(err).Fatalf("could not bind `%v` flags", cmd.CommandPath())
	}

	// set up logging
	logLevel := viper.GetString("log")
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

	// set up tracing
	if honeycombApiKey := viper.GetString("honeycomb-api-key"); honeycombApiKey != "" {
		if err := tracing.InitTracerWithHoneycomb(honeycombApiKey); err != nil {
			log.Fatal(err)
		}

		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			log.AllLevels[:log.GetLevel()+1]...,
		)))
	}

	// capture span in global variable to allow Execute() below to end it
	ctx, cmdSpan = tracing.Tracer().Start(ctx, fmt.Sprintf("CLI %v", cmd.CommandPath()), trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", tracedSettings())),
	))
	cmd.SetContext(ctx)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	formatter := new(log.TextFormatter)
	formatter.DisableTimestamp = true
	log.SetFormatter(formatter)

	// create a sub-scope to run deferred cleanups before shutting down the tracer
	err := func() error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		// Create a goroutine to watch for cancellation signals and aborting the
		// running command. Note that bubbletea converts ^C to a Quit message,
		// so we also need to handle that, but we still need to deal with the
		// regular signals.
		go func() {
			select {
			case signal := <-sigs:
				log.Info("Received signal, shutting down")
				if cmdSpan != nil {
					cmdSpan.SetAttributes(attribute.Bool("ovm.cli.aborted", true))
					cmdSpan.AddEvent("CLI Aborted", trace.WithAttributes(
						attribute.String("ovm.cli.signal", signal.String()),
					))
					cmdSpan.SetStatus(codes.Error, "CLI aborted by user")
				}
				cancel()
			case <-ctx.Done():
			}
		}()

		err := rootCmd.ExecuteContext(ctx)
		if err != nil {
			switch err := err.(type) { // nolint:errorlint // the selected error types are all top-level wrappers used by the CLI implementation
			case flagError:
				// print errors from viper with usage to stderr
				fmt.Fprintln(os.Stderr, err)
			case loggedError:
				log.WithContext(ctx).WithError(err.err).WithFields(err.fields).Error(err.message)
			}
			if cmdSpan != nil {
				// if printing the error was not requested by the appropriate
				// wrapper, only record the data to honeycomb and sentry, the
				// command already has handled logging
				cmdSpan.SetAttributes(
					attribute.Bool("ovm.cli.fatalError", true),
					attribute.String("ovm.cli.fatalError.msg", err.Error()),
				)
				cmdSpan.RecordError(err)
			}
			sentry.CaptureException(err)
		}

		return err
	}()

	// shutdown and submit any remaining otel data before exiting
	if cmdSpan != nil {
		cmdSpan.End()
	}
	tracing.ShutdownTracer()

	if err != nil {
		// If we have an error, exit with a non-zero status. Logging is handled by each command.
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

	width int
}

func (m authenticateModel) Init() tea.Cmd {
	return openBrowserCmd(m.deviceCode.VerificationURI)
}

func (m authenticateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

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
		m.status = Authenticated
		m.token = msg

	case statusMsg:
		switch msg {
		case PromptUser:
			cmds = append(cmds, openBrowserCmd(m.deviceCode.VerificationURI))
		case WaitingForConfirmation:
			m.status = WaitingForConfirmation
			cmds = append(cmds, awaitToken(m.ctx, m.config, m.deviceCode))
		case Authenticated:
		case ErrorAuthenticating:
		}

	case displayAuthorizationInstructionsMsg:
		m.status = WaitingForConfirmation
		cmds = append(cmds, awaitToken(m.ctx, m.config, m.deviceCode))

	case failedToAuthenticateErrorMsg:
		m.err = msg.err
		m.status = ErrorAuthenticating
		cmds = append(cmds, tea.Quit)

	case errMsg:
		m.err = msg.err
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

const beginAuthMessage string = `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

	%v

Then enter the code:

	%v
`

func (m authenticateModel) View() string {
	var output string

	switch m.status {
	case PromptUser, WaitingForConfirmation:
		prompt := fmt.Sprintf(beginAuthMessage, m.deviceCode.VerificationURI, m.deviceCode.UserCode)
		output = markdownToString(m.width, prompt)

	case Authenticated:
		output = wrap(RenderOk()+" Authenticated successfully. Press any key to continue.", m.width-4, 2)
	case ErrorAuthenticating:
		output = wrap(RenderErr()+" Unable to authenticate. Please try again.", m.width-4, 2)
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

type flagError struct {
	usage string
}

func (f flagError) Error() string {
	return f.usage
}

type loggedError struct {
	err     error
	fields  log.Fields
	message string
}

func (l loggedError) Error() string {
	return fmt.Sprintf("%v (%v): %v", l.message, l.fields, l.err)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return flagError{fmt.Sprintf("%v\n\n%s", err, c.UsageString())}
	})

	// General Config
	rootCmd.PersistentFlags().String("log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")
	cobra.CheckErr(viper.BindEnv("log", "OVERMIND_LOG", "LOG")) // fallback to global config

	// Support API Keys in the environment
	err := viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY")
	if err != nil {
		log.WithError(err).Fatal("could not bind api key to env")
	}

	// internal configs
	rootCmd.PersistentFlags().String("honeycomb-api-key", "hcaik_01j03qe0exnn2jxpj2vxkqb7yrqtr083kyk9rxxt2wzjamz8be94znqmwa", "If specified, configures opentelemetry libraries to submit traces to honeycomb.")
	rootCmd.PersistentFlags().String("sentry-dsn", "https://276b6d99c77358d9bf85aafbff81b515@o4504565700886528.ingest.us.sentry.io/4507413529690112", "If specified, configures the sentry libraries to send error reports to the service.")
	rootCmd.PersistentFlags().String("ovm-test-fake", "", "If non-empty, instructs some commands to only use fake data for fast development iteration.")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this command, 'release', 'debug' or 'test'. Defaults to 'release'.")

	// Mark these as hidden. This means that it will still be parsed of supplied,
	// and we will still look for it in the environment, but it won't be shown
	// in the help
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("honeycomb-api-key"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("sentry-dsn"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("ovm-test-fake"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("run-mode"))

	// Create groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "iac",
		Title: "Infrastructure as Code:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "api",
		Title: "Overmind API:",
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	replacer := strings.NewReplacer("-", "_")

	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match
}

func tracedSettings() map[string]any {
	result := make(map[string]any)
	result["log"] = viper.GetString("log")
	if viper.GetString("api-key") != "" {
		result["api-key"] = "[REDACTED]"
	}
	if viper.GetString("honeycomb-api-key") != "hcaik_01j03qe0exnn2jxpj2vxkqb7yrqtr083kyk9rxxt2wzjamz8be94znqmwa" {
		result["honeycomb-api-key"] = "[NON-DEFAULT]"
	}
	if viper.GetString("sentry-dsn") != "https://276b6d99c77358d9bf85aafbff81b515@o4504565700886528.ingest.us.sentry.io/4507413529690112" {
		result["sentry-dsn"] = "[NON-DEFAULT]"
	}
	result["ovm-test-fake"] = viper.GetString("ovm-test-fake")
	result["run-mode"] = viper.GetString("run-mode")
	result["timeout"] = viper.GetString("timeout")
	result["app"] = viper.GetString("app")
	result["change"] = viper.GetString("change")
	if viper.GetString("ticket-link") != "" {
		result["ticket-link"] = "[REDACTED]"
	}
	result["uuid"] = viper.GetString("uuid")

	return result
}

func login(ctx context.Context, cmd *cobra.Command, scopes []string) (context.Context, OvermindInstance, *oauth2.Token, error) {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		return ctx, OvermindInstance{}, nil, flagError{usage: fmt.Sprintf("invalid --timeout value '%v'\n\n%v", viper.GetString("timeout"), cmd.UsageString())}
	}

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		return ctx, OvermindInstance{}, nil, loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get instance data from app",
		}
	}

	ctx, token, err := ensureToken(ctx, oi, scopes)
	if err != nil {
		return ctx, OvermindInstance{}, nil, loggedError{
			err:     err,
			fields:  lf,
			message: "failed to authenticate",
		}
	}

	// apply a timeout to the main body of processing
	ctx, _ = context.WithTimeout(ctx, timeout) // nolint:govet // the context will not leak as the command will exit when it is done

	return ctx, oi, token, nil
}

func getAppUrl(frontend, app string) string {
	if frontend == "" && app == "" {
		return "https://app.overmind.tech"
	}
	if frontend != "" && app == "" {
		return frontend
	}
	if frontend != "" && app != "" {
		log.Warnf("Both --frontend and --app are set, but they are different. Using --app: %v", app)
	}
	return app
}
