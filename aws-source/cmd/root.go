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
	"github.com/overmindtech/cli/aws-source/proc"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aws-source",
	Short: "Remote primary source for AWS",
	Long: `This sources looks for AWS resources in your account.
`,
	Run: func(cmd *cobra.Command, args []string) {
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

		err := viper.UnmarshalKey("aws-regions", &awsAuthConfig.Regions)
		if err != nil {
			log.WithError(err).Fatal("Could not parse aws-regions")
		}

		engineConfig, err := discovery.EngineConfigFromViper("aws", tracing.Version())
		if err != nil {
			log.WithError(err).Fatal("Could not create engine config")
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
			log.WithError(err).Fatal("could not auth create clients")
		}

		rateLimitContext, rateLimitCancel := context.WithCancel(context.Background())
		defer rateLimitCancel()

		configs, err := proc.CreateAWSConfigs(awsAuthConfig)
		if err != nil {
			log.WithError(err).Fatal("Could not create AWS configs")
		}

		// Initialize the engine
		e, err := proc.InitializeAwsSourceEngine(
			rateLimitContext,
			engineConfig,
			999_999, // Very high max retries as it'll time out after 15min anyway
			configs...,
		)
		if err != nil {
			log.WithError(err).Fatal("Could not initialize AWS source")
		}

		// Start HTTP server for status
		healthCheckPath := "/healthz"

		http.HandleFunc(healthCheckPath, func(rw http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.Tracer().Start(r.Context(), "healthcheck")
			defer span.End()

			err := e.HealthCheck(ctx)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}

			fmt.Fprint(rw, "ok")
		})

		log.WithFields(log.Fields{
			"port": healthCheckPort,
			"path": healthCheckPath,
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
				"port": healthCheckPort,
				"path": healthCheckPath,
			}).Error("Could not start HTTP server for /healthz health checks")
		}()

		err = e.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Could not start engine")
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

		if err := tracing.InitTracerWithUpstreams("aws-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
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
