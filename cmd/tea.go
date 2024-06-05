package cmd

// this file contains a bunch of general helpers for building commands based on the bubbletea framework

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/aws-source/proc"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go/auth"
	stdlibsource "github.com/overmindtech/stdlib-source/sources"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

type OvermindCommandHandler func(ctx context.Context, args []string, oi OvermindInstance, token *oauth2.Token) error

type terraformStoredConfig struct {
	Config  string `json:"aws-config"`
	Profile string `json:"aws-profile"`
}

// viperGetApp fetches and validates the configured app url
func viperGetApp(ctx context.Context) (string, error) {
	app := viper.GetString("app")

	// Check to see if the URL is secure
	parsed, err := url.Parse(app)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --app")
		return "", fmt.Errorf("error parsing --app: %w", err)
	}

	if !(parsed.Scheme == "wss" || parsed.Scheme == "https" || parsed.Hostname() == "localhost") {
		return "", fmt.Errorf("target URL (%v) is insecure", parsed)
	}
	return app, nil
}

type FinalReportingModel interface {
	FinalReport() string
}

func CmdWrapper(action string, requiredScopes []string, commandModel func(args []string, parent *cmdModel, width int) tea.Model) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		// set up a context for the command
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cmdName := fmt.Sprintf("CLI %v", cmd.CommandPath())
		ctx, span := tracing.Tracer().Start(ctx, cmdName, trace.WithAttributes(
			attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
		))
		defer span.End()
		defer tracing.LogRecoverToExit(ctx, cmdName)

		// ensure that only error messages are printed to the console,
		// disrupting bubbletea rendering (and potentially getting overwritten).
		// Otherwise, when TEABUG is set, log to a file.
		if len(os.Getenv("TEABUG")) > 0 {
			f, err := tea.LogToFile("teabug.log", "debug")
			if err != nil {
				fmt.Println("fatal:", err)
				os.Exit(1)
			}
			// leave the log file open until the very last moment, so we capture everything
			// defer f.Close()
			log.SetOutput(f)
			formatter := new(log.TextFormatter)
			formatter.DisableTimestamp = false
			log.SetFormatter(formatter)
			viper.Set("log", "trace")
			log.SetLevel(log.TraceLevel)
		} else {
			// avoid log messages from sources and others to interrupt bubbletea rendering
			viper.Set("log", "fatal")
			log.SetLevel(log.FatalLevel)
		}

		// wrap the rest of the function in a closure to allow for cleaner error handling and deferring.
		err := func() error {
			timeout, err := time.ParseDuration(viper.GetString("timeout"))
			if err != nil {
				return fmt.Errorf("invalid --timeout value '%v', error: %w", viper.GetString("timeout"), err)
			}

			app, err := viperGetApp(ctx)
			if err != nil {
				return err
			}

			m := cmdModel{
				action:         action,
				ctx:            ctx,
				cancel:         cancel,
				timeout:        timeout,
				app:            app,
				requiredScopes: requiredScopes,
				apiKey:         viper.GetString("api-key"),
				tasks:          map[string]tea.Model{},
			}
			m.cmd = commandModel(args, &m, m.width)
			p := tea.NewProgram(&m)
			result, err := p.Run()
			if err != nil {
				return fmt.Errorf("could not start program: %w", err)
			}

			cmd, ok := result.(*cmdModel)
			if ok {
				frm, ok := cmd.cmd.(FinalReportingModel)
				if ok {
					fmt.Println(frm.FinalReport())
				}
			}

			return nil
		}()
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error running command")
			// don't forget that os.Exit() does not wait for telemetry to be flushed
			span.End()
			tracing.ShutdownTracer()
			os.Exit(1)
		}
	}
}

func InitializeSources(ctx context.Context, oi OvermindInstance, aws_config, aws_profile string, token *oauth2.Token) (func(), error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	natsNamePrefix := "overmind-cli"

	openapiUrl := *oi.ApiUrl
	openapiUrl.Path = "/api"
	tokenClient := auth.NewOAuthTokenClientWithContext(
		ctx,
		openapiUrl.String(),
		"",
		oauth2.StaticTokenSource(token),
	)

	natsOptions := auth.NATSOptions{
		NumRetries:        3,
		RetryDelay:        1 * time.Second,
		Servers:           []string{oi.NatsUrl.String()},
		ConnectionName:    fmt.Sprintf("%v.%v", natsNamePrefix, hostname),
		ConnectionTimeout: (10 * time.Second), // TODO: Make configurable
		MaxReconnects:     -1,
		ReconnectWait:     1 * time.Second,
		ReconnectJitter:   1 * time.Second,
		TokenClient:       tokenClient,
	}

	awsAuthConfig := proc.AwsAuthConfig{
		// TODO: ask user to select regions
		Regions: []string{"eu-west-1"},
	}

	switch aws_config {
	case "profile_input", "aws_profile":
		awsAuthConfig.Strategy = "sso-profile"
		awsAuthConfig.Profile = aws_profile
	case "defaults":
		awsAuthConfig.Strategy = "defaults"
	case "managed":
		// TODO: not implemented yet
	}

	awsEngine, err := proc.InitializeAwsSourceEngine(ctx, natsOptions, awsAuthConfig, 2_000)
	if err != nil {
		return func() {}, fmt.Errorf("failed to initialize AWS source engine: %w", err)
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = awsEngine.Start()
	if err != nil {
		return func() {}, fmt.Errorf("failed to start AWS source engine: %w", err)
	}

	stdlibEngine, err := stdlibsource.InitializeEngine(natsOptions, 2_000, true)
	if err != nil {
		return func() {
			_ = awsEngine.Stop()
		}, fmt.Errorf("failed to initialize stdlib source engine: %w", err)
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = stdlibEngine.Start()
	if err != nil {
		return func() {
			_ = awsEngine.Stop()
		}, fmt.Errorf("failed to start stdlib source engine: %w", err)
	}

	return func() {
		_ = awsEngine.Stop()
		_ = stdlibEngine.Stop()
	}, nil
}
