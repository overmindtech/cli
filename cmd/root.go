package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/go/auth"
	"github.com/overmindtech/cli/go/cliauth"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "overmind",
	Short: "The Overmind CLI",
	Long: `Calculate the blast radius of your changes, track risks, and make changes with
confidence.

This CLI will prompt you for authentication using Overmind's OAuth service,
however it can also be configured to use an API key by setting the OVM_API_KEY
environment variable.`,
	Version:      tracing.Version(),
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
		if err := tracing.InitTracerWithUpstreams("overmind-cli", honeycombApiKey, ""); err != nil {
			log.Fatal(err)
		}

		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			log.AllLevels[:log.GetLevel()+1]...,
		)))
	}
	// set up app, it may be ambiguous if frontend is set
	app := getAppUrl(viper.GetString("frontend"), viper.GetString("app"))
	if app == "" {
		log.Fatal("no app specified, please use --app or set the 'APP' environment variable")
	}
	viper.Set("app", app)
	// capture span in global variable to allow Execute() below to end it
	ctx, cmdSpan = tracing.Tracer().Start(ctx, fmt.Sprintf("CLI %v", cmd.CommandPath()), trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", tracedSettings())),
	))
	cmd.SetContext(ctx)

	// Check for CLI version updates (non-blocking with timeout)
	// Run in goroutine to avoid blocking command execution
	// Use command context so the check is cancelled when command completes
	currentVersion := tracing.Version()
	go displayVersionWarning(ctx, currentVersion)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	formatter := new(log.TextFormatter)
	formatter.DisableTimestamp = true
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)

	// Configure pterm to output to stderr instead of stdout
	// This ensures status messages don't interfere with piped output
	pterm.SetDefaultOutput(os.Stderr)
	pterm.Info.Writer = os.Stderr
	pterm.Success.Writer = os.Stderr
	pterm.Warning.Writer = os.Stderr
	pterm.Error.Writer = os.Stderr

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
			switch err := err.(type) { //nolint:errorlint // the selected error types are all top-level wrappers used by the CLI implementation
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
	tracing.ShutdownTracer(context.Background())

	if err != nil {
		// If we have an error, exit with a non-zero status. Logging is handled by each command.
		os.Exit(1)
	}
}

// ptermLogger adapts pterm output to the cliauth.Logger interface
type ptermLogger struct{}

func (p *ptermLogger) Info(msg string, keysAndValues ...any) {
	if len(keysAndValues) > 0 {
		kvs := make([]string, 0, len(keysAndValues)/2)
		for i := 0; i+1 < len(keysAndValues); i += 2 {
			kvs = append(kvs, fmt.Sprintf("%v: %v", keysAndValues[i], keysAndValues[i+1]))
		}
		pterm.Info.Println(fmt.Sprintf("%s (%s)", msg, strings.Join(kvs, ", ")))
	} else {
		pterm.Info.Println(msg)
	}
}

func (p *ptermLogger) Error(msg string, keysAndValues ...any) {
	if len(keysAndValues) > 0 {
		kvs := make([]string, 0, len(keysAndValues)/2)
		for i := 0; i+1 < len(keysAndValues); i += 2 {
			kvs = append(kvs, fmt.Sprintf("%v: %v", keysAndValues[i], keysAndValues[i+1]))
		}
		pterm.Error.Println(fmt.Sprintf("%s (%s)", msg, strings.Join(kvs, ", ")))
	} else {
		pterm.Error.Println(msg)
	}
}

// getChangeUUIDAndCheckStatus returns the UUID of a change, as selected by --uuid or --change, or a change with the specified status and having --ticket-link
func getChangeUUIDAndCheckStatus(ctx context.Context, oi sdp.OvermindInstance, expectedStatus sdp.ChangeStatus, ticketLink string, errorOnNotFound bool) (uuid.UUID, error) {
	var changeUUID uuid.UUID
	var err error

	uuidString := viper.GetString("uuid")
	changeUrlString := viper.GetString("change")

	// If no arguments are specified then return an error
	if uuidString == "" && changeUrlString == "" && ticketLink == "" {
		return uuid.Nil, errors.New("no change specified; use one of --change, --ticket-link or --uuid")
	}

	// Check UUID first if more than one is set
	if uuidString != "" {
		changeUUID, err = uuid.Parse(uuidString)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid --uuid value '%v', error: %w", uuidString, err)
		}
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.change.uuid", changeUUID.String()),
		)
		return changeUUID, nil
	}

	// Then check for a change URL
	if changeUrlString != "" {
		uuidFromChangeURL, err := parseChangeUrl(changeUrlString)
		if err != nil {
			return uuidFromChangeURL, err
		}
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.change.uuid", uuidFromChangeURL.String()),
		)
		return uuidFromChangeURL, nil
	}

	// Finally look up by ticket link with retry
	changeUUID, err = getChangeByTicketLinkWithRetry(ctx, oi, ticketLink, expectedStatus, errorOnNotFound)
	if errorOnNotFound && err != nil {
		return uuid.Nil, err
	}
	// this could be uuid.Nil if the change is not found and errorOnNotFound is false
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("ovm.change.uuid", changeUUID.String()),
	)
	return changeUUID, nil
}

// getChangeUUID resolves a change UUID from --uuid, --change, or --ticket-link without
// checking the change status. Use this when the server-side RPC handles status validation
// (e.g. EndChangeSimple already validates status atomically and has queuing logic).
func getChangeUUID(ctx context.Context, oi sdp.OvermindInstance, ticketLink string) (uuid.UUID, error) {
	uuidString := viper.GetString("uuid")
	changeUrlString := viper.GetString("change")

	// If no arguments are specified then return an error
	if uuidString == "" && changeUrlString == "" && ticketLink == "" {
		return uuid.Nil, errors.New("no change specified; use one of --change, --ticket-link or --uuid")
	}

	// Check UUID first if more than one is set
	if uuidString != "" {
		changeUUID, err := uuid.Parse(uuidString)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid --uuid value '%v', error: %w", uuidString, err)
		}
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.change.uuid", changeUUID.String()),
		)
		return changeUUID, nil
	}

	// Then check for a change URL
	if changeUrlString != "" {
		uuidFromChangeURL, err := parseChangeUrl(changeUrlString)
		if err != nil {
			return uuidFromChangeURL, err
		}
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.change.uuid", uuidFromChangeURL.String()),
		)
		return uuidFromChangeURL, nil
	}

	// Finally look up by ticket link (single attempt, no status check)
	client := AuthenticatedChangesClient(ctx, oi)
	change, err := client.GetChangeByTicketLink(ctx, &connect.Request[sdp.GetChangeByTicketLinkRequest]{
		Msg: &sdp.GetChangeByTicketLinkRequest{
			TicketLink: ticketLink,
		},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("error looking up change with ticket link %v: %w", ticketLink, err)
	}

	uuidPtr := change.Msg.GetChange().GetMetadata().GetUUIDParsed()
	if uuidPtr == nil {
		return uuid.Nil, fmt.Errorf("change found with ticket link %v but has no UUID", ticketLink)
	}

	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("ovm.change.uuid", uuidPtr.String()),
	)
	return *uuidPtr, nil
}

// getChangeByTicketLinkWithRetry performs the GetChangeByTicketLink API call with retry logic,
// retrying both on error and when the status does not match the expected status.
// NB api-server will only return the latest change with this ticket link.
func getChangeByTicketLinkWithRetry(ctx context.Context, oi sdp.OvermindInstance, ticketLink string, expectedStatus sdp.ChangeStatus, errorOnNotFound bool) (uuid.UUID, error) {
	client := AuthenticatedChangesClient(ctx, oi)

	var change *connect.Response[sdp.GetChangeResponse]
	var currentStatus sdp.ChangeStatus
	var err error
	maxRetries := 3
	if !errorOnNotFound {
		// If not erroring on not found, only attempt once.
		maxRetries = 1
	}
	retryDelay := 3 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		change, err = client.GetChangeByTicketLink(ctx, &connect.Request[sdp.GetChangeByTicketLinkRequest]{
			Msg: &sdp.GetChangeByTicketLinkRequest{
				TicketLink: ticketLink,
			},
		})
		if err == nil {
			// change found
			var uuidPtr *uuid.UUID
			if change != nil && change.Msg != nil && change.Msg.GetChange() != nil && change.Msg.GetChange().GetMetadata() != nil {
				uuidPtr = change.Msg.GetChange().GetMetadata().GetUUIDParsed()
				currentStatus = change.Msg.GetChange().GetMetadata().GetStatus()
				if uuidPtr != nil && (currentStatus == expectedStatus) {
					// Success: we have a UUID and status matches the expected status
					return *uuidPtr, nil
				}
			}
		}
		// Log the error and retry if not the last attempt
		if attempt < maxRetries {
			logFields := log.Fields{
				"ovm.change.ticketLink": ticketLink,
				"expectedStatus":        expectedStatus.String(),
				"attempt":               attempt,
				"maxRetries":            maxRetries,
				"currentStatus":         currentStatus.String(),
			}
			if err != nil {
				logFields["error"] = err.Error()
				log.WithContext(ctx).WithFields(logFields).Debug("failed to get change by ticket link, retrying")
			} else {
				log.WithContext(ctx).WithFields(logFields).Debug("change found but status does not match, retrying")
			}
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		// Final attempt failed with an error
		return uuid.Nil, fmt.Errorf("error looking up change with ticket link %v after %d attempts: %w", ticketLink, maxRetries, err)
	}
	// Final attempt found a change but status did not match
	return uuid.Nil, fmt.Errorf("change %s found with ticket link %v. Change status %v does not match expected status %v after %d attempts", change.Msg.GetChange().GetMetadata().GetUUIDParsed(), ticketLink, currentStatus.String(), expectedStatus.String(), maxRetries)
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

	// Initialize the pallette for lip gloss, it detects the colour of the terminal.
	InitPalette()

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
	ctx, _ = context.WithTimeout(ctx, timeout) //nolint:govet,gosec // the context will not leak as the command will exit when it is done

	return ctx, oi, token, nil
}

func ensureToken(ctx context.Context, oi sdp.OvermindInstance, requiredScopes []string) (context.Context, *oauth2.Token, error) {
	apiKey := viper.GetString("api-key")
	app := viper.GetString("app")

	token, err := cliauth.GetToken(ctx, oi, app, apiKey, requiredScopes, &ptermLogger{})
	if err != nil {
		return ctx, nil, fmt.Errorf("error getting token: %w", err)
	}
	if token == nil {
		return ctx, nil, fmt.Errorf("error token: nil")
	}

	// Add account/auth info to the span for traceability
	tok, err := josejwt.ParseSigned(token.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return ctx, nil, fmt.Errorf("Error running program: received invalid token: %w", err)
	}
	out := josejwt.Claims{}
	customClaims := auth.CustomClaims{}
	err = tok.UnsafeClaimsWithoutVerification(&out, &customClaims)
	if err != nil {
		return ctx, nil, fmt.Errorf("Error running program: received unparsable token: %w", err)
	}
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.Bool("ovm.auth.authenticated", true),
		attribute.String("ovm.auth.accountName", customClaims.AccountName),
		attribute.String("ovm.auth.scopes", customClaims.Scope),
		attribute.String("ovm.auth.subject", out.Subject),
		attribute.String("ovm.auth.expiry", out.Expiry.Time().String()),
	)

	ok, missing, err := cliauth.HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return ctx, nil, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return ctx, nil, fmt.Errorf("authenticated successfully, but you don't have the required permission: '%v'", missing)
	}

	// Store the token for later use by sdp-go's auth client. Note that this
	// loses access to the RefreshToken and could be done better by using an
	// oauth2.TokenSource, but this would require more work on updating sdp-go
	// that is currently not scheduled.
	ctx = context.WithValue(ctx, auth.UserTokenContextKey{}, token.AccessToken)

	return ctx, token, nil
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
