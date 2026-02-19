package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/logging"
	"github.com/overmindtech/cli/go/tracing"
	"github.com/overmindtech/cli/sources/snapshot/adapters"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "snapshot-source",
	Short:        "Discovery source that serves data from a snapshot file",
	SilenceUsage: true,
	Long: `Snapshot source loads a snapshot from a file or URL and responds to 
discovery queries with items from that snapshot. This enables local testing 
with fixed data and deterministic re-runs of v6 investigations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		defer tracing.LogRecoverToReturn(ctx, "snapshot-source.root")

		// Get snapshot source (required)
		snapshotSource := viper.GetString("snapshot-source")
		if snapshotSource == "" {
			return fmt.Errorf("snapshot-source is required (use --snapshot-source or SNAPSHOT_SOURCE env var)")
		}

		log.WithField("snapshot-source", snapshotSource).Info("Starting snapshot source")

		// Get engine config
		engineConfig, err := discovery.EngineConfigFromViper("snapshot", tracing.Version())
		if err != nil {
			log.WithError(err).Error("Could not get engine config from viper")
			return fmt.Errorf("could not get engine config from viper: %w", err)
		}

		// Create a basic engine first
		e, err := discovery.NewEngine(engineConfig)
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not create engine")
			return fmt.Errorf("could not create engine: %w", err)
		}

		// Start HTTP server for health checks before initialization
		healthCheckPort := viper.GetInt("health-check-port")
		e.ServeHealthProbes(healthCheckPort)

		// Start the engine (NATS connection) before adapter init so heartbeats work
		err = e.Start(ctx)
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not start engine")
			return fmt.Errorf("could not start engine: %w", err)
		}

		// Snapshot adapters load from files/URLs which may fail, so we use
		// the initialization pattern with error handling
		err = adapters.InitializeAdapters(ctx, e, snapshotSource)
		if err != nil {
			initErr := fmt.Errorf("could not initialize snapshot adapters: %w", err)
			log.WithError(initErr).Error("Snapshot source initialization failed - pod will stay running with error status")
			e.SetInitError(initErr)
			sentry.CaptureException(initErr)
		} else {
			e.StartSendingHeartbeats(ctx)
		}

		<-ctx.Done()

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

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once to
// the rootCmd.
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

	// General config options
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/srcman/config/source.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")
	cobra.CheckErr(viper.BindEnv("log", "SNAPSHOT_LOG", "LOG")) // fallback to global config

	// Snapshot-specific config
	rootCmd.PersistentFlags().String("snapshot-source", "", "Path to snapshot file or URL to load (required). Can be a local file path or http(s) URL.")
	cobra.CheckErr(viper.BindEnv("snapshot-source", "SNAPSHOT_SOURCE", "SNAPSHOT_PATH", "SNAPSHOT_URL"))

	// engine config options
	discovery.AddEngineFlags(rootCmd)

	rootCmd.PersistentFlags().IntP("health-check-port", "", 8089, "The port that the health check should run on")
	cobra.CheckErr(viper.BindEnv("health-check-port", "SNAPSHOT_HEALTH_CHECK_PORT", "HEALTH_CHECK_PORT", "SNAPSHOT_SERVICE_PORT", "SERVICE_PORT")) // new names + backwards compat

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	cobra.CheckErr(viper.BindEnv("honeycomb-api-key", "SNAPSHOT_HONEYCOMB_API_KEY", "HONEYCOMB_API_KEY")) // fallback to global config
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	cobra.CheckErr(viper.BindEnv("sentry-dsn", "SNAPSHOT_SENTRY_DSN", "SENTRY_DSN")) // fallback to global config
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", true, "Set to false to emit logs as text for easier reading in development.")
	cobra.CheckErr(viper.BindEnv("json-log", "SNAPSHOT_SOURCE_JSON_LOG", "JSON_LOG")) // fallback to global config

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
			log.WithError(bindErr).Error("Could not bind flag to viper")
			return fmt.Errorf("could not bind flag to viper: %w", bindErr)
		}

		if viper.GetBool("json-log") {
			logging.ConfigureLogrusJSON(log.StandardLogger())
		}

		if err := tracing.InitTracerWithUpstreams("snapshot-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
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
	// Do not set env prefix so APP, API_KEY, NATS_* etc. are read the same as other sources (aws, gcp).
	// Snapshot-specific options use explicit BindEnv (e.g. SNAPSHOT_SOURCE) in flag init.
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
