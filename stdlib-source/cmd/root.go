package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/stdlib-source/adapters"
	"github.com/overmindtech/cli/tracing"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "stdlib-source",
	Short: "Standard library of remotely accessible items",
	Long: `Gets details of items that are globally scoped
(usually) and able to be queried without authentication.
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer tracing.LogRecoverToReturn(ctx, "stdlib-source.root")

		// get engine config
		engineConfig, err := discovery.EngineConfigFromViper("stdlib", tracing.Version())
		if err != nil {
			log.WithError(err).Fatal("Could not get engine config from viper")
		}
		reverseDNS := viper.GetBool("reverse-dns")

		log.WithFields(log.Fields{
			"reverse-dns": reverseDNS,
		}).Info("Got config")

		// Validate the auth params and create a token client if we are using
		// auth
		err = engineConfig.CreateClients()
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Fatal("could not create auth clients")
		}

		e, err := adapters.InitializeEngine(
			engineConfig,
			reverseDNS,
		)
		if err != nil {
			log.WithError(err).Error("Could not initialize aws source")
			return
		}

		// Start HTTP server for status
		healthCheckPort := viper.GetString("service-port")
		healthCheckPath := "/healthz"

		healthCheckDNSAdapter := adapters.DNSAdapter{}

		// Set up the health check
		healthCheck := func(ctx context.Context) error {
			if !e.IsNATSConnected() {
				return errors.New("NATS not connected")
			}

			// We have seen some issues with DNS lookups within kube where the
			// stdlib container will just start timing out on DNS requests. We
			// should check that the DNS adapter is working so that the
			// container can die if this happens to it
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			_, err := healthCheckDNSAdapter.Search(ctx, "global", "www.google.com", true)
			if err != nil {
				return fmt.Errorf("test dns lookup failed: %w", err)
			}

			return nil
		}

		if e.EngineConfig.HeartbeatOptions != nil {
			e.EngineConfig.HeartbeatOptions.HealthCheck = healthCheck
		}
		http.HandleFunc(healthCheckPath, func(rw http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.Tracer().Start(r.Context(), "healthcheck")
			defer span.End()

			err := healthCheck(ctx)
			if err == nil {
				fmt.Fprint(rw, "ok")
			} else {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
			}
		})

		log.WithFields(log.Fields{
			"port": healthCheckPort,
			"path": healthCheckPath,
		}).Debug("Starting healthcheck server")

		go func() {
			defer sentry.Recover()

			server := &http.Server{
				Addr:    fmt.Sprintf(":%v", healthCheckPort),
				Handler: nil,
				// due to https://github.com/securego/gosec/pull/842
				ReadTimeout:  5 * time.Second, // Set the read timeout to 5 seconds
				WriteTimeout: 5 * time.Second, // Set the write timeout to 5 seconds
			}

			err := server.ListenAndServe()

			log.WithError(err).WithFields(log.Fields{
				"port": healthCheckPort,
				"path": healthCheckPath,
			}).Error("Could not start HTTP server for /healthz health checks")
		}()

		err = e.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Could not start engine")

			os.Exit(1)
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

			os.Exit(1)
		}

		log.Info("Stopped")

		os.Exit(0)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.add
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

	// General config options
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/srcman/config/source.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")
	cobra.CheckErr(viper.BindEnv("log", "STDLIB_LOG", "LOG")) // fallback to global config
	rootCmd.PersistentFlags().Bool("reverse-dns", false, "If true, will perform reverse DNS lookups on IP addresses")

	// engine config options
	discovery.AddEngineFlags(rootCmd)

	rootCmd.PersistentFlags().String("service-port", "8089", "the port to listen on")
	cobra.CheckErr(viper.BindEnv("service-port", "STDLIB_SERVICE_PORT", "SERVICE_PORT")) // fallback to srcman config
	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	cobra.CheckErr(viper.BindEnv("honeycomb-api-key", "STDLIB_HONEYCOMB_API_KEY", "HONEYCOMB_API_KEY")) // fallback to global config
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	cobra.CheckErr(viper.BindEnv("sentry-dsn", "STDLIB_SENTRY_DSN", "SENTRY_DSN")) // fallback to global config
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")

	// Bind these to viper
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not bind flags to viper")
	}

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
				err = viper.BindPFlag(f.Name, f)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Fatal("Could not bind flag to viper")
				}
			}
		})

		if err := tracing.InitTracerWithUpstreams("stdlib-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
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
	viper.SetEnvPrefix("STDLIB")
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
