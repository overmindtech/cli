package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/huh"
	awssource "github.com/overmindtech/aws-source/cmd"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go/auth"
	stdlibsource "github.com/overmindtech/stdlib-source/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

// terraformPlanCmd represents the `terraform plan` command
var terraformPlanCmd = &cobra.Command{
	Use:   "plan [terraform options...]",
	Short: "Creates a new Change from a given terraform plan file",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `terraform plan` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goroutine to watch for cancellation signals
		go func() {
			select {
			case <-sigs:
				cancel()
			case <-ctx.Done():
			}
		}()

		exitcode := TerraformPlan(ctx, args, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func TerraformPlan(ctx context.Context, files []string, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI TerraformPlan", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	hostname, err := os.Hostname()
	if err != nil {
		log.WithError(err).Fatal("Could not determine hostname for use in NATS connection name")
	}

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get instance data from app")
		return 1
	}

	ctx, token, err := ensureToken(ctx, oi, []string{"changes:write"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	_, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	r := NewTermRenderer()

	// TODO: store this in the api-server and skip questioning the user after the first time
	aws_config := "aborted"
	options := []huh.Option[string]{}
	aws_profile := os.Getenv("AWS_PROFILE")
	if aws_profile != "" {
		options = append(options,
			huh.NewOption(fmt.Sprintf("Use $AWS_PROFILE (currently: '%v')", aws_profile), "aws_profile"),
			huh.NewOption("Use a different profile", "profile_input"),
		)
	} else {
		options = append(options,
			huh.NewOption("Use the default settings", "defaults"),
			huh.NewOption("Use an AWS SSO profile", "profile_input"),
		)
	}
	// TODO: what URL needs to get opened here?
	// TODO: how to wait for a source to be configured?
	// options = append(options,
	// 	huh.NewOption("Run managed source (opens browser)", "managed"),
	// )
	aws_config_select := huh.NewSelect[string]().
		Title("Choose how to access your AWS account (read-only):").
		Options(options...).
		Value(&aws_config).
		WithAccessible(accessibleMode)
	err = aws_config_select.Run()
	// annoyingly, huh doesn't leave the form on screen - except in
	// accessible mode, so this prints it again so the scrollback looks
	// sensible
	if !accessibleMode {
		fmt.Println(aws_config_select.View())
	}
	if err != nil {
		fmt.Printf("Aborting: %v\n", err)
		return 1
	}

	awsAuthConfig := awssource.AwsAuthConfig{
		Regions: []string{"eu-west-1"},
	}

	switch aws_config {
	case "profile_input":
		aws_profile_input := huh.NewInput().
			Title("Input the name of the AWS profile to use:").
			Value(&aws_profile).
			WithAccessible(accessibleMode)
		err = aws_profile_input.Run()
		// annoyingly, huh doesn't leave the form on screen - except in
		// accessible mode, so this prints it again so the scrollback looks
		// sensible
		if !accessibleMode {
			fmt.Println(aws_profile_input.View())
		}
		if err != nil {
			fmt.Printf("Aborting: %v\n", err)
			return 1
		}
		// reset the environment to the requested value
		os.Setenv("AWS_PROFILE", aws_profile)
		awsAuthConfig.Strategy = "sso-profile"
		awsAuthConfig.Profile = aws_profile
	case "aws_profile":
		// can continue with the existing config
		awsAuthConfig.Strategy = "sso-profile"
		awsAuthConfig.Profile = aws_profile
	case "defaults":
		// just continue
		awsAuthConfig.Strategy = "defaults"
	case "managed":
		// TODO: not implemented yet
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

	awsEngine, err := awssource.InitializeAwsSourceEngine(natsOptions, awsAuthConfig, 2_000)
	if err != nil {
		log.WithError(err).Error("failed to initialize AWS source engine")
		return 1
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = awsEngine.Start()
	if err != nil {
		log.WithError(err).Error("failed to start AWS source engine")
		return 1
	}
	defer func() {
		_ = awsEngine.Stop()
	}()

	stdlibEngine, err := stdlibsource.InitializeStdlibSourceEngine(natsOptions, 2_000, true)
	if err != nil {
		log.WithError(err).Error("failed to initialize stdlib source engine")
		return 1
	}

	// todo: pass in context with timeout to abort timely and allow Ctrl-C to work
	err = stdlibEngine.Start()
	if err != nil {
		log.WithError(err).Error("failed to start stdlib source engine")
		return 1
	}
	defer func() {
		_ = stdlibEngine.Stop()
	}()

	prompt := `# Doing something

NATS connection: %v

* AWS Source: running
* stdlib Source: running

This will be doing something: %vAWS_PROFILE=%v terraform plan -out overmind_plan.out%v
`
	out, err := r.Render(fmt.Sprintf(prompt, awsEngine.IsNATSConnected(), "`", aws_profile, "`"))
	if err != nil {
		panic(err)
	}
	fmt.Print(out)

	return 0
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
}
