package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/durationpb"
)

var cfgFile string
var logLevel string

var minStatusInterval = durationpb.New(250 * time.Millisecond)

//go:generate sh -c "echo -n $(git describe --long) > commit.txt"
//go:embed commit.txt
var cliVersion string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "ovm-cli",
	Short:   "A CLI to interact with the overmind API",
	Long:    `The ovm-cli allows direct access to the overmind API`,
	Version: cliVersion,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `root` flags")
		}
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

// ensureToken
func ensureToken(ctx context.Context, signals chan os.Signal) (context.Context, error) {
	// get a token from the api key if present
	if viper.GetString("api-key") != "" {
		log.WithContext(ctx).Debug("using provided token for authentication")
		apiKey := viper.GetString("api-key")
		if strings.HasPrefix(apiKey, "ovm_api_") {
			// exchange api token for JWT
			client := UnauthenticatedApiKeyClient(ctx)
			resp, err := client.ExchangeKeyForToken(ctx, &connect.Request[sdp.ExchangeKeyForTokenRequest]{
				Msg: &sdp.ExchangeKeyForTokenRequest{
					ApiKey: apiKey,
				},
			})
			if err != nil {
				return ctx, fmt.Errorf("error authenticating the API token: %w", err)
			}
			log.WithContext(ctx).Debug("successfully authenticated")
			apiKey = resp.Msg.AccessToken
		} else {
			return ctx, errors.New("--api-key does not match pattern 'ovm_api_*'")
		}
		return context.WithValue(ctx, sdp.UserTokenContextKey{}, apiKey), nil
	}

	// Check to see if the URL is secure
	gatewayUrl := viper.GetString("gateway-url")
	if gatewayUrl == "" {
		gatewayUrl = fmt.Sprintf("%v/api/gateway", viper.GetString("url"))
		viper.Set("gateway-url", gatewayUrl)
	}
	parsed, err := url.Parse(gatewayUrl)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --url")
		return ctx, fmt.Errorf("error parsing --gateway-url: %w", err)
	}

	if parsed.Scheme == "wss" || parsed.Scheme == "https" || parsed.Hostname() == "localhost" {
		// Authenticate using the oauth resource owner password flow
		config := oauth2.Config{
			ClientID: viper.GetString("auth0-client-id"),
			Scopes:   []string{"openid", "profile", "email", "gateway:stream", "request:send", "reverselink:request", "account:read", "source:read", "source:write", "api:read", "api:write", "gateway:objects"},
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
			log.WithContext(ctx).Debug("Received interrupt, exiting")
			return ctx, errors.New("cancelled")
		}

		// Stop the server
		err = srv.Shutdown(ctx)
		if err != nil {
			log.WithContext(ctx).WithError(err).Info("failed to shutdown auth callback server, but continuing anyways")
		}

		// Set the token
		return context.WithValue(ctx, sdp.UserTokenContextKey{}, token.AccessToken), nil
	}
	return ctx, fmt.Errorf("no --api-key configured and target URL (%v) is insecure", parsed)
}

func init() {
	cobra.OnInitialize(initConfig)

	// General Config
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is redacted.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")

	// api endpoint
	rootCmd.PersistentFlags().String("url", "https://api.prod.overmind.tech", "The overmind API endpoint")
	rootCmd.PersistentFlags().String("gateway-url", "", "The overmind Gateway endpoint (defaults to /api/gateway on --url)")

	// authorization
	rootCmd.PersistentFlags().String("api-key", "", "The API key to use for authentication, also read from OVM_API_KEY environment variable")
	err := viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY")
	if err != nil {
		log.WithError(err).Fatal("could not bind token")
	}
	rootCmd.PersistentFlags().String("api-key-url", "", "The overmind API Keys endpoint (defaults to --url)")
	rootCmd.PersistentFlags().String("auth0-client-id", "j3LylZtIosVPZtouKI8WuVHmE6Lluva1", "OAuth Client ID to use when connecting with auth")
	rootCmd.PersistentFlags().String("auth0-domain", "om-prod.eu.auth0.com", "Auth0 domain to connect to")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", false, "Set to true to emit logs as json for easier parsing")

	// debugging
	rootCmd.PersistentFlags().Bool("stdout-trace-dump", false, "Dump all otel traces to stdout for debugging")

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
