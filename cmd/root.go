package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"golang.org/x/oauth2"
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

// ensureToken
func ensureToken(ctx context.Context, signals chan os.Signal) error {
	// Check to see if the URL is secure
	gatewayURL, err := url.Parse(viper.GetString("url"))
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --url")
		return err
	}

	if viper.GetString("token") == "" && (gatewayURL.Scheme == "wss" || gatewayURL.Scheme == "https" || gatewayURL.Hostname() == "localhost") {
		// Authenticate using the oauth resource owner password flow
		config := oauth2.Config{
			ClientID: viper.GetString("auth0-client-id"),
			Scopes:   []string{"gateway:stream", "request:send", "reverselink:request", "account:read", "source:read", "source:write", "api:read", "api:write", "gateway:objects"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%v/authorize", viper.GetString("auth0-domain")),
				TokenURL: fmt.Sprintf("https://%v/oauth/token", viper.GetString("auth0-domain")),
			},
			RedirectURL: "http://127.0.0.1:7837/oauth/callback",
		}

		tokenChan := make(chan *oauth2.Token, 1)
		// create a random token for this exchange
		oAuthStateString := uuid.New().String()

		// Start the web server to listen for the callback
		handler := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			queryParts, _ := url.ParseQuery(r.URL.RawQuery)

			// Use the authorization code that is pushed to the redirect
			// URL.
			code := queryParts["code"][0]
			log.WithContext(ctx).Debugf("Got code: %v", code)

			state := queryParts["state"][0]
			log.WithContext(ctx).Debugf("Got state: %v", state)

			if state != oAuthStateString {
				log.WithContext(ctx).Errorf("Invalid state, expected %v, got %v", oAuthStateString, state)
			}

			// Exchange will do the handshake to retrieve the initial access token.
			log.WithContext(ctx).Debug("Exchanging code for token")
			tok, err := config.Exchange(ctx, code)
			if err != nil {
				log.WithContext(ctx).Error(err)
				return
			}
			log.WithContext(ctx).Debug("Got token 1!")

			tokenChan <- tok

			// show success page
			msg := "<p><strong>Success!</strong></p>"
			msg = msg + "<p>You are authenticated and can now return to the CLI.</p>"
			fmt.Fprint(w, msg)
		}

		audienceOption := oauth2.SetAuthURLParam("audience", "https://api.overmind.tech")

		u := config.AuthCodeURL(oAuthStateString, oauth2.AccessTypeOnline, audienceOption)

		log.WithContext(ctx).Infof("Log in here: %v", u)

		// Start the webserver
		log.WithContext(ctx).Trace("Starting webserver to listen for callback, press Ctrl+C to cancel")
		srv := &http.Server{Addr: ":7837"}
		http.HandleFunc("/oauth/callback", handler)

		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error. port in use?
				log.WithContext(ctx).Errorf("HTTP Server error: %v", err)
			}
		}()

		// Wait for the token or cancel
		var token *oauth2.Token
		select {
		case token = <-tokenChan:
			log.WithContext(ctx).Debug("Got token 2!")
		case <-signals:
			log.WithContext(ctx).Info("Received interrupt, exiting")
			return nil
		}

		// Stop the server
		srv.Shutdown(ctx)

		// Set the token
		viper.Set("token", token.AccessToken)
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// General Config
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is redacted.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")

	// api endpoint
	rootCmd.PersistentFlags().String("url", "https://api.prod.overmind.tech/", "The overmind API endpoint")

	// authorization
	rootCmd.PersistentFlags().String("auth0-client-id", "j3LylZtIosVPZtouKI8WuVHmE6Lluva1", "OAuth Client ID to use when connecting with auth")
	rootCmd.PersistentFlags().String("auth0-domain", "om-prod.eu.auth0.com", "Auth0 domain to connect to")
	rootCmd.PersistentFlags().String("token", "", "The token to use for authentication")
	viper.BindEnv("token", "OVM_TOKEN", "TOKEN")

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
