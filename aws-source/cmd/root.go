package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/aws-source/proc"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/logging"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "aws-source",
	Short:        "Remote primary source for AWS",
	SilenceUsage: true,
	Long: `This sources looks for AWS resources in your account.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer tracing.LogRecoverToReturn(ctx, "aws-source.root")
		healthCheckPort := viper.GetInt("health-check-port")

		awsAuthConfig := proc.AwsAuthConfig{
			Strategy:        viper.GetString("aws-access-strategy"),
			AccessKeyID:     viper.GetString("aws-access-key-id"),
			SecretAccessKey: viper.GetString("aws-secret-access-key"),
			ExternalID:      viper.GetString("aws-external-id"),
			TargetRoleARN:   viper.GetString("aws-target-role-arn"),
			Profile:         viper.GetString("aws-profile"),
			AutoConfig:      viper.GetBool("auto-config"),
		}

		// Parse regions early so we can detect config errors before adapter init
		var regionParseErr error
		if err := viper.UnmarshalKey("aws-regions", &awsAuthConfig.Regions); err != nil {
			regionParseErr = fmt.Errorf("could not parse aws-regions: %w", err)
			log.WithError(err).Error("Could not parse aws-regions")
		}

		engineConfig, err := discovery.EngineConfigFromViper("aws", tracing.Version())
		if err != nil {
			log.WithError(err).Error("Could not create engine config")
			return fmt.Errorf("could not create engine config: %w", err)
		}

		log.WithFields(log.Fields{
			"aws-regions":         awsAuthConfig.Regions,
			"aws-access-strategy": awsAuthConfig.Strategy,
			"aws-external-id":     awsAuthConfig.ExternalID,
			"aws-target-role-arn": awsAuthConfig.TargetRoleARN,
			"aws-profile":         awsAuthConfig.Profile,
			"auto-config":         awsAuthConfig.AutoConfig,
			"health-check-port":   healthCheckPort,
		}).Info("Got config")

		err = engineConfig.CreateClients()
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("could not auth create clients")
		}

		rateLimitContext, rateLimitCancel := context.WithCancel(context.Background())
		defer rateLimitCancel()

		// Create a basic engine first so we can serve health probes and heartbeats even if init fails
		e, err := discovery.NewEngine(engineConfig)
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not create engine")
		}

		// Serve health probes before initialization so they're available even on failure
		e.ServeHealthProbes(healthCheckPort)

		// If region parsing failed, surface the original error instead of letting
		// CreateAWSConfigs fail later with a misleading "no regions specified" message.
		if regionParseErr != nil {
			log.WithError(regionParseErr).Error("AWS source initialization failed - pod will stay running with error status")
			e.SetInitError(regionParseErr)
			sentry.CaptureException(regionParseErr)
		} else if configs, configErr := proc.CreateAWSConfigs(awsAuthConfig); configErr != nil {
			// Don't exit - store error, serve probes, send heartbeats
			initErr := fmt.Errorf("could not create AWS configs: %w", configErr)
			log.WithError(initErr).Error("AWS source initialization failed - pod will stay running with error status")
			e.SetInitError(initErr)
			sentry.CaptureException(initErr)
		} else {
			// Initialize the AWS adapters
			adapterErr := proc.InitializeAwsSourceAdapters(
				rateLimitContext,
				e,
				999_999, // Very high max retries as it'll time out after 15min anyway
				configs...,
			)
			if adapterErr != nil {
				// Don't exit - store error, serve probes, send heartbeats
				initErr := fmt.Errorf("could not initialize AWS source adapters: %w", adapterErr)
				log.WithError(initErr).Error("AWS source initialization failed - pod will stay running with error status")
				e.SetInitError(initErr)
				sentry.CaptureException(initErr)
			}
		}

		// Start the engine regardless of initialization success
		// This ensures NATS connection and heartbeats work even if adapters failed to initialize
		err = e.Start(ctx)
		if err != nil {
			// Only fatal errors during engine start should cause exit (NATS connection, etc.)
			log.WithError(err).Error("Could not start engine")
			return fmt.Errorf("could not start engine: %w", err)
		}

		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		<-sigs

		log.Info("Stopping engine")

		err = e.Stop()

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Could not stop engine")

			return fmt.Errorf("could not stop engine: %w", err)
		}
		log.Info("Stopped")

		return nil
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

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	var logLevel string

	// add engine flags
	discovery.AddEngineFlags(rootCmd)

	// General config options
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/srcman/config/source.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")

	// Custom flags for this source
	rootCmd.PersistentFlags().String("aws-access-strategy", "defaults", "The strategy to use to access this customer's AWS account. Valid values: 'access-key', 'external-id', 'sso-profile', 'defaults'. Default: 'defaults'.")
	rootCmd.PersistentFlags().String("aws-access-key-id", "", "The ID of the access key to use")
	rootCmd.PersistentFlags().String("aws-secret-access-key", "", "The secret access key to use for auth")
	rootCmd.PersistentFlags().String("aws-external-id", "", "The external ID to use when assuming the customer's role")
	rootCmd.PersistentFlags().String("aws-target-role-arn", "", "The role to assume in the customer's account")
	rootCmd.PersistentFlags().String("aws-profile", "", "The AWS SSO Profile to use. Defaults to $AWS_PROFILE, then whatever the AWS SDK's SSO config defaults to")
	rootCmd.PersistentFlags().String("aws-regions", "", "Comma-separated list of AWS regions that this source should operate in")
	rootCmd.PersistentFlags().BoolP("auto-config", "a", false, "Use the local AWS config, the same as the AWS CLI could use. This can be set up with \"aws configure\"")
	rootCmd.PersistentFlags().IntP("health-check-port", "", 8080, "The port that the health check should run on")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", true, "Set to false to emit logs as text for easier reading in development.")
	cobra.CheckErr(viper.BindEnv("json-log", "AWS_SOURCE_JSON_LOG", "JSON_LOG"))

	// Bind these to viper
	cobra.CheckErr(viper.BindPFlags(rootCmd.PersistentFlags()))

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if lvl, err := log.ParseLevel(logLevel); err == nil {
			log.SetLevel(lvl)
		} else {
			log.SetLevel(log.InfoLevel)
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Could not parse log level")
		}

		log.AddHook(TerminationLogHook{})

		// Bind flags that haven't been set to the values from viper of we have them
		var bindErr error
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			// Bind the flag to viper only if it has a non-empty default
			if f.DefValue != "" || f.Changed {
				if err := viper.BindPFlag(f.Name, f); err != nil {
					bindErr = err
				}
			}
		})
		if bindErr != nil {
			log.WithError(bindErr).Error("could not bind flag to viper")
			return fmt.Errorf("could not bind flag to viper: %w", bindErr)
		}

		if viper.GetBool("json-log") {
			logging.ConfigureLogrusJSON(log.StandardLogger())
		}

		if err := tracing.InitTracerWithUpstreams("aws-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
			log.WithError(err).Error("could not init tracer")
			return fmt.Errorf("could not init tracer: %w", err)
		}
		return nil
	}
	// shut down tracing at the end of the process
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		tracing.ShutdownTracer(context.Background())
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigFile(cfgFile)

	replacer := strings.NewReplacer("-", "_")

	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Infof("Using config file: %v", viper.ConfigFileUsed())
	}
}

// TerminationLogHook A hook that logs fatal errors to the termination log
type TerminationLogHook struct{}

func (t TerminationLogHook) Levels() []log.Level {
	return []log.Level{log.FatalLevel}
}

func (t TerminationLogHook) Fire(e *log.Entry) error {
	// shutdown tracing first to ensure all spans are flushed
	tracing.ShutdownTracer(context.Background())
	tLog, err := os.OpenFile("/dev/termination-log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	var message string

	message = e.Message

	for k, v := range e.Data {
		message = fmt.Sprintf("%v %v=%v", message, k, v)
	}

	_, err = tLog.WriteString(message)

	return err
}
