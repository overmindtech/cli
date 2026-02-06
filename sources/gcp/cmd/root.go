package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/logging"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/proc"
	"github.com/overmindtech/cli/tracing"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "gcp-source",
	Short:        "Remote primary source for GCP",
	SilenceUsage: true,
	Long: `This sources looks for GCP resources in your account.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer tracing.LogRecoverToReturn(ctx, "gcp-source.root")
		healthCheckPort := viper.GetInt("health-check-port")

		engineConfig, err := discovery.EngineConfigFromViper("gcp", tracing.Version())
		if err != nil {
			log.WithError(err).Error("Could not create engine config")
			return fmt.Errorf("could not create engine config: %w", err)
		}

		err = engineConfig.CreateClients()
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("could not auth create clients")
		}

		// Create a basic engine first so we can serve health probes and heartbeats even if init fails
		e, err := discovery.NewEngine(engineConfig)
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not create engine")
		}

		// Serve health probes before initialization so they're available even on failure
		e.ServeHealthProbes(healthCheckPort)

		// Try to initialize GCP adapters
		// If this fails, we'll store the error and continue running with no adapters
		err = proc.InitializeAdapters(ctx, e, nil)
		if err != nil {
			// Don't exit - store error, serve probes, send heartbeats
			initErr := fmt.Errorf("could not initialize GCP source adapters: %w", err)
			log.WithError(initErr).Error("GCP source initialization failed - pod will stay running with error status")
			e.SetInitError(initErr)
			sentry.CaptureException(initErr)
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
			log.WithError(err).Error("Could not stop engine")
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
	rootCmd.PersistentFlags().IntP("health-check-port", "", 8080, "The port that the health check should run on")
	rootCmd.PersistentFlags().String("gcp-parent", "", "GCP parent resource to discover from. Can be an organization (organizations/{org_id}), folder (folders/{folder_id}), or project (project-id or projects/{project_id}). If not specified, all accessible projects will be discovered automatically. Format examples: 'organizations/123456789012', 'folders/123456789012', 'my-project-id', 'projects/my-project-id'")
	rootCmd.PersistentFlags().String("gcp-project-id", "", "(Deprecated: use --gcp-parent instead) GCP Project ID that this source should operate in. If not specified, all accessible projects will be discovered automatically using the Cloud Resource Manager API. Requires 'resourcemanager.projects.list' permission (included in 'roles/browser' role).")
	rootCmd.PersistentFlags().String("gcp-regions", "", "Comma-separated list of GCP regions that this source should operate in")
	rootCmd.PersistentFlags().String("gcp-zones", "", "Comma-separated list of GCP zones that this source should operate in")
	rootCmd.PersistentFlags().String("gcp-impersonation-service-account-email", "", "The email of the service account to impersonate. Leave empty for direct access using Application Default Credentials.")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", true, "Set to false to emit logs as text for easier reading in development.")
	cobra.CheckErr(viper.BindEnv("json-log", "GCP_SOURCE_JSON_LOG", "JSON_LOG"))

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

		if err := tracing.InitTracerWithUpstreams("gcp-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
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
	tLog, err := os.OpenFile("/dev/termination-log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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
