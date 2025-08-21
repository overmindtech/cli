package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
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
	Use:   "gcp-source",
	Short: "Remote primary source for GCP",
	Long: `This sources looks for GCP resources in your account.
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer tracing.LogRecoverToReturn(ctx, "gcp-source.root")
		healthCheckPort := viper.GetInt("health-check-port")

		engineConfig, err := discovery.EngineConfigFromViper("gcp", tracing.Version())
		if err != nil {
			log.WithError(err).Fatal("Could not create engine config")
		}

		err = engineConfig.CreateClients()
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Fatal("could not auth create clients")
		}

		e, err := proc.Initialize(ctx, engineConfig, nil)
		if err != nil {
			log.WithError(err).Fatal("Could not initialize GCP source")
		}

		e.StartSendingHeartbeats(ctx)

		// Start HTTP server for status
		healthCheckPath := "/healthz"

		http.HandleFunc(healthCheckPath, func(rw http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.HealthCheckTracer().Start(r.Context(), "healthcheck")
			defer span.End()

			err := e.HealthCheck(ctx)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}

			fmt.Fprint(rw, "ok")
		})

		log.WithFields(log.Fields{
			"ovm.source.type": "gcp",
			"ovm.source.port": healthCheckPort,
			"ovm.source.path": healthCheckPath,
		}).Debug("Starting healthcheck server")

		go func() {
			defer sentry.Recover()

			server := &http.Server{
				Addr:         fmt.Sprintf(":%v", healthCheckPort),
				Handler:      nil,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			err := server.ListenAndServe()

			log.WithError(err).WithFields(log.Fields{
				"ovm.source.type": "gcp",
				"ovm.source.port": healthCheckPort,
				"ovm.source.path": healthCheckPath,
			}).Error("Could not start HTTP server for /healthz health checks")
		}()

		err = e.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"ovm.source.type":  "gcp",
				"ovm.source.error": err,
			}).Fatal("Could not start engine")
		}

		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		<-sigs

		log.Info("Stopping engine")

		err = e.Stop()

		if err != nil {
			log.WithFields(log.Fields{
				"ovm.source.type":  "gcp",
				"ovm.source.error": err,
			}).Error("Could not stop engine")

			os.Exit(1)
		}
		log.Info("Stopped")

		os.Exit(0)
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
	rootCmd.PersistentFlags().String("gcp-regions", "", "Comma-separated list of GCP regions that this source should operate in")
	rootCmd.PersistentFlags().String("gcp-zones", "", "Comma-separated list of GCP zones that this source should operate in")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")

	// Bind these to viper
	cobra.CheckErr(viper.BindPFlags(rootCmd.PersistentFlags()))

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
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
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			// Bind the flag to viper only if it has a non-empty default
			if f.DefValue != "" || f.Changed {
				err := viper.BindPFlag(f.Name, f)
				if err != nil {
					log.WithError(err).Fatal("could not bind flag to viper")
				}
			}
		})

		if err := tracing.InitTracerWithUpstreams("gcp-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
			log.Fatal(err)
		}
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
