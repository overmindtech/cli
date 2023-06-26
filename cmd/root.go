package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/overmindtech/ovm-cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
)

var cfgFile string
var logLevel string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ovm-cli",
	Short: "A CLI to interact with the overmind API",
	Long:  `The ovm-cli allows direct access to the overmind API`,
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

	// General Config
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is redacted.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", false, "Set to true to emit logs as json for easier parsing")

	// debugging
	rootCmd.PersistentFlags().Bool("stdout-trace-dump", false, "Dump all otel traces to stdout for debugging")

	viper.BindPFlags(rootCmd.PersistentFlags())

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Read env vars
		logLevel = viper.GetString("log")
		lvl, err := log.ParseLevel(logLevel)
		if err != nil {
			log.WithFields(log.Fields{"level": logLevel, "err": err}).Errorf("couldn't parse `log` config, defaulting to `info`")
			lvl = log.InfoLevel
		}
		log.SetLevel(lvl)
		log.WithField("level", lvl).Infof("set log level from config")

		if viper.GetBool("json-log") {
			log.SetFormatter(&log.JSONFormatter{})
		}

		if err := tracing.InitTracerWithHoneycomb(viper.GetString("honeycomb-api-key")); err != nil {
			log.Fatal(err)
		}

		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		)))
	}
	// shut down tracing at the end of the process
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		tracing.ShutdownTracer()
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
	} else {
		log.WithFields(log.Fields{"err": err}).Errorf("Error reading config file")
	}
}
