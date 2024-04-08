package cmd

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
		log.WithError(err).Fatal("could not initialize instance-data fetch")
	}

	req = req.WithContext(ctx)
	log.WithField("instanceDataUrl", instanceDataUrl).Debug("Fetching instance-data")
	res, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		log.WithError(err).Fatal("could not fetch instance-data")
	}

	if res.StatusCode != 200 {
		log.WithField("status-code", res.StatusCode).Fatal("instance-data fetch returned non-200 status")
	}

	defer res.Body.Close()
	data := instanceData{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		log.WithError(err).Fatal("could not parse instance-data")
	}

	instance.ApiUrl, err = url.Parse(data.Api)
	if err != nil {
		return instance, fmt.Errorf("invalid api_url value '%v' in instance-data, error: %w", data.Api, err)
	}
	instance.NatsUrl, err = url.Parse(data.Nats)
	if err != nil {
		return instance, fmt.Errorf("invalid nats_url value '%v' in instance-data, error: %w", data.Nats, err)
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

// extracts custom claims from a JWT token. Note that this does not verify the
// signature of the token, it just extracts the claims from the payload
func extractClaims(token string) (*sdp.CustomClaims, error) {
	// We aren't interested in checking the signature of the token since
	// the server will do that. All we need to do is make sure it
	// contains the right scopes. Therefore we just parse the payload
	// directly
	sections := strings.Split(token, ".")

	if len(sections) != 3 {
		return nil, errors.New("token is not a JWT")
	}

	// Decode the payload
	decodedPayload, err := base64.RawURLEncoding.DecodeString(sections[1])

	if err != nil {
		return nil, fmt.Errorf("error decoding token payload: %w", err)
	}

	// Parse the payload
	claims := new(sdp.CustomClaims)

	err = json.Unmarshal(decodedPayload, claims)

	if err != nil {
		return nil, fmt.Errorf("error parsing token payload: %w", err)
	}

	return claims, nil
}

// reads the locally cached token if it exists and is valid returns the token,
// its scopes, and an error if any. The scopes are returned even if they are
// insufficient to allow cached tokens to be added to rather than constantly
// replaced
func readLocalToken(homeDir string, requiredScopes []string) (*oauth2.Token, []string, error) {
	// Read in the token JSON file
	path := filepath.Join(homeDir, ".overmind", "token.json")

	token := new(oauth2.Token)

	// Check that the file exists
	if _, err := os.Stat(path); err != nil {
		return nil, nil, err
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening token file at %v: %w", path, err)
	}

	// Decode the file
	err = json.NewDecoder(file).Decode(token)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding token file at %v: %w", path, err)
	}

	// Check to see if the token is still valid
	if !token.Valid() {
		return nil, nil, errors.New("token is no longer valid")
	}

	claims, err := extractClaims(token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting claims from token: %w", err)
	}
	if claims.Scope == "" {
		return nil, nil, errors.New("token does not have any scopes")
	}

	currentScopes := strings.Split(claims.Scope, " ")

	// Check that we actually got the claims we asked for.
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return nil, currentScopes, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return nil, currentScopes, fmt.Errorf("local token is missing this permission: '%v'", missing)
	}

	log.Debugf("Using local token from %v", path)
	return token, currentScopes, nil
}

// Gets a token using an API key
func getAPIKeyToken(ctx context.Context, oi OvermindInstance, apiKey string, requiredScopes []string) (*oauth2.Token, error) {
	log.WithContext(ctx).Debug("using provided token for authentication")

	var token *oauth2.Token

	if !strings.HasPrefix(apiKey, "ovm_api_") {
		return nil, errors.New("OVM_API_KEY does not match pattern 'ovm_api_*'")
	}

	// exchange api token for JWT
	client := UnauthenticatedApiKeyClient(ctx, oi)
	resp, err := client.ExchangeKeyForToken(ctx, &connect.Request[sdp.ExchangeKeyForTokenRequest]{
		Msg: &sdp.ExchangeKeyForTokenRequest{
			ApiKey: apiKey,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error authenticating the API token: %w", err)
	}
	log.WithContext(ctx).Debug("successfully got a token from the API key")

	token = &oauth2.Token{
		AccessToken: resp.Msg.GetAccessToken(),
		TokenType:   "Bearer",
	}

	// Check that we actually got the claims we asked for. If you don't have
	// permission auth0 will just not assign those scopes rather than fail
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return nil, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("authenticated successfully, but your API key is missing this permission: '%v'", missing)
	}

	return token, nil
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
	return openBrowser(m.deviceCode.VerificationURI)
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
			return m, openBrowser(m.deviceCode.VerificationURI)
		case WaitingForConfirmation:
			m.status = WaitingForConfirmation
			return m, awaitToken(m.ctx, m.config, m.deviceCode)
		case Authenticated:
		case ErrorAuthenticating:
			{
				return m, nil
			}
		}

	case browserOpenErrorMsg:
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
		sp := createSpinner()
		output += sp.View() + " Waiting for confirmation..."
	case Authenticated:
		output = "✅ Authenticated successfully. Press any key to continue."
	case ErrorAuthenticating:
		output = "⛔️ Unable to authenticate. Try again."
	}

	return containerStyle.Render(output)
}

type errMsg struct{ err error }
type browserOpenErrorMsg struct{ err error }
type failedToAuthenticateErrorMsg struct{ err error }

func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		err := browser.OpenURL(url)
		if err != nil {
			return browserOpenErrorMsg{err}
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

// Gets a token from Oauth with the required scopes. This method will also cache
// that token locally for use later, and will use the cached token if possible
func getOauthToken(ctx context.Context, oi OvermindInstance, requiredScopes []string) (*oauth2.Token, error) {
	var localScopes []string

	// Check for a locally saved token in ~/.overmind
	if home, err := os.UserHomeDir(); err == nil {
		var localToken *oauth2.Token

		localToken, localScopes, err = readLocalToken(home, requiredScopes)

		if err != nil {
			log.WithContext(ctx).Debugf("Error reading local token, ignoring: %v", err)
		} else {
			// If we already have the right scopes, return the token
			return localToken, nil
		}
	}

	// If we need to get a new token, request the required scopes on top of
	// whatever ones the current local, valid token has so that we don't
	// keep replacing it

	// Check to see if the URL is secure
	appurl := viper.GetString("app")
	parsed, err := url.Parse(appurl)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --app")
		return nil, fmt.Errorf("error parsing --app: %w", err)
	}

	if !(parsed.Scheme == "wss" || parsed.Scheme == "https" || parsed.Hostname() == "localhost") {
		return nil, fmt.Errorf("target URL (%v) is insecure", parsed)
	}
	// If we need to get a new token, request the required scopes on top of
	// whatever ones the current local, valid token has so that we don't
	// keep replacing it
	requestScopes := append(requiredScopes, localScopes...)

	// Authenticate using the oauth device authorization flow
	config := oauth2.Config{
		ClientID: viper.GetString("cli-auth0-client-id"),
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", viper.GetString("cli-auth0-domain")),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", viper.GetString("cli-auth0-domain")),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", viper.GetString("cli-auth0-domain")),
		},
		Scopes: requestScopes,
	}

	deviceCode, err := config.DeviceAuth(ctx,
		oauth2.SetAuthURLParam("audience", oi.Audience),
		oauth2.AccessTypeOffline,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting device code: %w", err)
	}

	m := authenticateModel{ctx: ctx, status: PromptUser, deviceCode: deviceCode, config: config}
	authenticateProgram := tea.NewProgram(m)

	result, err := authenticateProgram.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	m, ok := result.(authenticateModel)
	if !ok {
		fmt.Println("Error running program: result is not authenticateModel")
		os.Exit(1)
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Bool("ovm.cli.authenticated", true))

	// Save the token locally
	if home, err := os.UserHomeDir(); err == nil {
		// Create the directory if it doesn't exist
		err = os.MkdirAll(filepath.Join(home, ".overmind"), 0700)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Failed to create ~/.overmind directory")
		}

		// Write the token to a file
		path := filepath.Join(home, ".overmind", "token.json")
		file, err := os.Create(path)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to create token file at %v", path)
		}

		// Encode the token
		err = json.NewEncoder(file).Encode(m.token)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to encode token file at %v", path)
		}

		log.WithContext(ctx).Debugf("Saved token to %v", path)
	}

	return m.token, nil
}

// ensureToken gets a token from the environment or from the user, and returns a
// context holding the token tthat can be used by sdp-go's helper functions to
// authenticate against the API
func ensureToken(ctx context.Context, oi OvermindInstance, requiredScopes []string) (context.Context, *oauth2.Token, error) {
	var token *oauth2.Token
	var err error

	// get a token from the api key if present
	if apiKey := viper.GetString("api-key"); apiKey != "" {
		token, err = getAPIKeyToken(ctx, oi, apiKey, requiredScopes)
	} else {
		token, err = getOauthToken(ctx, oi, requiredScopes)
	}
	if err != nil {
		return ctx, nil, fmt.Errorf("error getting token: %w", err)
	}

	// Check that we actually got the claims we asked for. If you don't have
	// permission auth0 will just not assign those scopes rather than fail
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return ctx, nil, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return ctx, nil, fmt.Errorf("authenticated successfully, but you don't have the required permission: '%v'", missing)
	}

	// store the token for later use by sdp-go's auth client. Note that this
	// loses access to the RefreshToken and could be done better by using an
	// oauth2.TokenSource, but this would require more work on updating sdp-go
	// that is currently not scheduled
	ctx = context.WithValue(ctx, sdp.UserTokenContextKey{}, token.AccessToken)

	return ctx, token, nil
}

// Returns whether a set of claims has all of the required scopes. It also
// accounts for when a user has write access but required read access, they
// aren't the same but the user will have access anyway so this will pass
//
// Returns true if the token has the required scopes. Otherwise, false and the missing permission for displaying or logging
func HasScopesFlexible(token *oauth2.Token, requiredScopes []string) (bool, string, error) {
	if token == nil {
		return false, "", errors.New("HasScopesFlexible: token is nil")
	}

	claims, err := extractClaims(token.AccessToken)
	if err != nil {
		return false, "", fmt.Errorf("error extracting claims from token: %w", err)
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			// If they don't have the *exact* scope, check to see if they have
			// write access to the same service
			sections := strings.Split(scope, ":")
			var hasWriteInstead bool

			if len(sections) == 2 {
				service, action := sections[0], sections[1]

				if action == "read" {
					hasWriteInstead = claims.HasScope(fmt.Sprintf("%v:write", service))
				}
			}

			if !hasWriteInstead {
				return false, scope, nil
			}
		}
	}

	return true, "", nil
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

	// Support API Keys in the environment
	err := viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY")
	if err != nil {
		log.WithError(err).Fatal("could not bind api key to env")
	}

	// internal configs
	rootCmd.PersistentFlags().String("cli-auth0-client-id", "QMfjMww3x4QTpeXiuRtMV3JIQkx6mZa4", "OAuth Client ID to use when connecting with auth0")
	rootCmd.PersistentFlags().String("cli-auth0-domain", "om-prod.eu.auth0.com", "Auth0 domain to connect to")
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb. This requires --otel to be set.")

	// Mark these as hidden. This means that it will still be parsed of supplied,
	// and we will still look for it in the environment, but it won't be shown
	// in the help
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("cli-auth0-client-id"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("cli-auth0-domain"))
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("honeycomb-api-key"))

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
			lvl = log.InfoLevel
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
