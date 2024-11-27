package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/sdp-go"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

//go:generate sh -c "echo -n $(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD) > commit.txt"
//go:embed commit.txt
var cliVersion string

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

const beginAuthMessage string = `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

	%v

Then enter the code:

	%v
`

// getChangeUuid returns the UUID of a change, as selected by --uuid or --change, or a state with the specified status and having --ticket-link
func getChangeUuid(ctx context.Context, oi sdp.OvermindInstance, expectedStatus sdp.ChangeStatus, ticketLink string, errNotFound bool) (uuid.UUID, error) {
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

func login(ctx context.Context, cmd *cobra.Command, scopes []string, writer io.Writer) (context.Context, sdp.OvermindInstance, *oauth2.Token, error) {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		return ctx, sdp.OvermindInstance{}, nil, flagError{usage: fmt.Sprintf("invalid --timeout value '%v'\n\n%v", viper.GetString("timeout"), cmd.UsageString())}
	}

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	var multi *pterm.MultiPrinter
	if writer == nil {
		multi = pterm.DefaultMultiPrinter.WithWriter(os.Stderr)
		_, _ = multi.Start()
	} else {
		multi = pterm.DefaultMultiPrinter.WithWriter(writer)
	}

	connectSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Connecting to Overmind")

	oi, err := sdp.NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		connectSpinner.Fail("Failed to get instance data from app")
		_, _ = multi.Stop()
		return ctx, sdp.OvermindInstance{}, nil, loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get instance data from app",
		}
	}

	connectSpinner.Success("Connected to Overmind")
	_, _ = multi.Stop()

	ctx, token, err := ensureToken(ctx, oi, scopes)
	if err != nil {
		connectSpinner.Fail("Failed to authenticate")
		return ctx, sdp.OvermindInstance{}, nil, loggedError{
			err:     err,
			fields:  lf,
			message: "failed to authenticate",
		}
	}

	// apply a timeout to the main body of processing
	ctx, _ = context.WithTimeout(ctx, timeout) // nolint:govet // the context will not leak as the command will exit when it is done

	return ctx, oi, token, nil
}

func ensureToken(ctx context.Context, oi sdp.OvermindInstance, requiredScopes []string) (context.Context, *oauth2.Token, error) {
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

// Gets a token from Oauth with the required scopes. This method will also cache
// that token locally for use later, and will use the cached token if possible
func getOauthToken(ctx context.Context, oi sdp.OvermindInstance, requiredScopes []string) (*oauth2.Token, error) {
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
	requestScopes := append(requiredScopes, localScopes...)

	// Authenticate using the oauth device authorization flow
	config := oauth2.Config{
		ClientID: oi.CLIClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", oi.Auth0Domain),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", oi.Auth0Domain),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", oi.Auth0Domain),
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

	var token *oauth2.Token
	var urlToOpen string
	if deviceCode.VerificationURIComplete != "" {
		urlToOpen = deviceCode.VerificationURIComplete
	} else {
		urlToOpen = deviceCode.VerificationURI
	}

	_ = browser.OpenURL(urlToOpen) // nolint:errcheck // we don't care if the browser fails to open
	pterm.Print(
		markdownToString(MAX_TERMINAL_WIDTH, fmt.Sprintf(
			beginAuthMessage,
			deviceCode.VerificationURI,
			deviceCode.UserCode,
		)))

	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()

	authSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Waiting for browser authentication")

	token, err = config.DeviceAccessToken(ctx, deviceCode)
	if err != nil {
		authSpinner.Fail("Unable to authenticate. Please try again.")
		log.WithContext(ctx).WithError(err).Error("Error getting device code")
		os.Exit(1)
	}

	if token == nil {
		authSpinner.Fail("Error running program: no token received")
		log.WithContext(ctx).Error("Error running program: no token received")
		os.Exit(1)
	}

	authSpinner.Success("Authenticated successfully")
	_, _ = multi.Stop()

	tok, err := josejwt.ParseSigned(token.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		pterm.Error.Printf("Error running program: received invalid token: %v", err)
		os.Exit(1)
	}
	out := josejwt.Claims{}
	customClaims := sdp.CustomClaims{}
	err = tok.UnsafeClaimsWithoutVerification(&out, &customClaims)
	if err != nil {
		pterm.Error.Printf("Error running program: received unparsable token: %v", err)
		os.Exit(1)
	}

	if cmdSpan != nil {
		cmdSpan.SetAttributes(
			attribute.Bool("ovm.cli.authenticated", true),
			attribute.String("ovm.cli.accountName", customClaims.AccountName),
			attribute.String("ovm.cli.userId", out.Subject),
		)
	}

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
		err = json.NewEncoder(file).Encode(token)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to encode token file at %v", path)
		}

		log.WithContext(ctx).Debugf("Saved token to %v", path)
	}

	return token, nil
}

// Gets a token using an API key
func getAPIKeyToken(ctx context.Context, oi sdp.OvermindInstance, apiKey string, requiredScopes []string) (*oauth2.Token, error) {
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
