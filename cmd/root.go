package cmd

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"golang.org/x/oauth2"
)

const Auth0ClientId = "j3LylZtIosVPZtouKI8WuVHmE6Lluva1"
const Auth0Domain = "om-prod.eu.auth0.com"

var logLevel string

//go:generate sh -c "echo -n $(git describe --tags --long) > commit.txt"
//go:embed commit.txt
var cliVersion string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "overmind",
	Short: "The Overmind CLI",
	Long: `Calculate the blast radius of your changes, track risks, and make changes with
confidence.

This CLI will prompt you for authentication using Overmind's OAuth service,
however it can also be configured to use an API key by setting the OVM_API_KEY
environment variable.`,
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

// extracts custom claims from a JWT token. Note that this does not verify the
// signature of the token, it just extracts the claims from the payload
func extractClaims(token string) (*sdp.CustomClaims, error) {
	// We aren't interested in checking the signature of the token since
	// the server will do that. All we need to do is make sure it
	// contains the right scopes. Therefore we just parse the payload
	// directly
	sections := strings.Split(token, ".")

	if len(sections) != 3 {
		return nil, errors.New("token is not a JWT")
	}

	// Decode the payload
	decodedPayload, err := base64.RawURLEncoding.DecodeString(sections[1])

	if err != nil {
		return nil, fmt.Errorf("error decoding token payload: %w", err)
	}

	// Parse the payload
	claims := new(sdp.CustomClaims)

	err = json.Unmarshal(decodedPayload, claims)

	if err != nil {
		return nil, fmt.Errorf("error parsing token payload: %w", err)
	}

	return claims, nil
}

// reads the locally cached token if it exists and is valid returns the token,
// its scopes, and an error if any. The scopes are returned even if they are
// insufficient to allow cached tokens to be added to rather than constantly
// replaced
func readLocalToken(homeDir string, expectedScopes []string) (string, []string, error) {
	// Read in the token JSON file
	path := filepath.Join(homeDir, ".overmind", "token.json")

	token := new(oauth2.Token)

	// Check that the file exists
	if _, err := os.Stat(path); err != nil {
		return "", nil, err
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("error opening token file at %v: %w", path, err)
	}

	// Decode the file
	err = json.NewDecoder(file).Decode(token)
	if err != nil {
		return "", nil, fmt.Errorf("error decoding token file at %v: %w", path, err)
	}

	// Check to see if the token is still valid
	if !token.Valid() {
		return "", nil, errors.New("token is no longer valid")
	}

	claims, err := extractClaims(token.AccessToken)

	if err != nil {
		return "", nil, fmt.Errorf("error extracting claims from token: %w", err)
	}

	if claims.Scope == "" {
		return "", nil, errors.New("token does not have any scopes")
	}

	currentScopes := strings.Split(claims.Scope, " ")

	// Check that the token has the right scopes
	for _, scope := range expectedScopes {
		if !claims.HasScope(scope) {
			return "", currentScopes, fmt.Errorf("token does not have required scope '%v'", scope)
		}
	}

	return token.AccessToken, currentScopes, nil
}

// ensureToken
func ensureToken(ctx context.Context, requiredScopes []string) (context.Context, error) {
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
			apiKey = resp.Msg.GetAccessToken()
		} else {
			return ctx, errors.New("OVM_API_KEY does not match pattern 'ovm_api_*'")
		}
		return context.WithValue(ctx, sdp.UserTokenContextKey{}, apiKey), nil
	}

	var localScopes []string

	// Check for a locally saved token in ~/.overmind
	if home, err := os.UserHomeDir(); err == nil {
		var localToken string

		localToken, localScopes, err = readLocalToken(home, requiredScopes)

		if err != nil {
			log.WithContext(ctx).Debugf("Error reading local token, ignoring: %v", err)
		} else {
			return context.WithValue(ctx, sdp.UserTokenContextKey{}, localToken), nil
		}
	}

	// Check to see if the URL is secure
	appurl := viper.GetString("url")
	parsed, err := url.Parse(appurl)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to parse --url")
		return ctx, fmt.Errorf("error parsing --url: %w", err)
	}

	if parsed.Scheme == "wss" || parsed.Scheme == "https" || parsed.Hostname() == "localhost" {
		// If we need to get a new token, request the required scopes on top of
		// whatever ones the current local, valid token has so that we don't
		// keep replacing it
		requestScopes := append(requiredScopes, localScopes...)

		// Authenticate using the oauth resource owner password flow
		config := oauth2.Config{
			ClientID: Auth0ClientId,
			Scopes:   requestScopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://%v/authorize", Auth0Domain),
				TokenURL: fmt.Sprintf("https://%v/oauth/token", Auth0Domain),
			},
			RedirectURL: "http://127.0.0.1:7837/oauth/callback",
		}

		tokenChan := make(chan *oauth2.Token, 1)
		// create a random token for this exchange
		oAuthStateString := uuid.New().String()

		// Start the web server to listen for the callback
		handler := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			queryParts, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				log.WithContext(ctx).WithError(err).WithFields(log.Fields{
					"url": r.URL,
				}).Error("Failed to parse url")
			}

			// Use the authorization code that is pushed to the redirect
			// URL.
			code := queryParts["code"][0]
			log.WithContext(ctx).Debugf("Got code: %v", code)

			state := queryParts["state"][0]
			log.WithContext(ctx).Debugf("Got state: %v", state)

			if state != oAuthStateString {
				log.WithContext(ctx).Errorf("Invalid state, expected %v, got %v", oAuthStateString, state)
				return
			}

			// Exchange will do the handshake to retrieve the initial access token.
			log.WithContext(ctx).Debug("Exchanging code for token")
			tok, err := config.Exchange(ctx, code)
			if err != nil {
				log.WithContext(ctx).Error(err)
				return
			}
			log.WithContext(ctx).Debug("Got token")

			tokenChan <- tok

			// show success page
			msg := "<p><strong>Success!</strong></p>"
			msg = msg + "<p>You are authenticated and can now return to the CLI.</p>"
			fmt.Fprint(w, msg)
		}

		audienceOption := oauth2.SetAuthURLParam("audience", "https://api.overmind.tech")

		u := config.AuthCodeURL(oAuthStateString, oauth2.AccessTypeOnline, audienceOption)
		log.WithContext(ctx).Infof("Follow this link to authenticate: %v", Underline.TextStyle(u))

		// Start the webserver
		log.WithContext(ctx).Trace("Starting webserver to listen for callback, press Ctrl+C to cancel")
		srv := &http.Server{Addr: ":7837", ReadHeaderTimeout: 30 * time.Second}
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
			// Keep working
		case <-ctx.Done():
			return ctx, ctx.Err()
		}

		// Stop the server
		err = srv.Shutdown(ctx)
		if err != nil {
			log.WithContext(ctx).WithError(err).Warn("failed to shutdown auth callback server, but continuing anyway")
		}

		// Check that we actually got the claims we asked for. If you don't have
		// permission auth0 will just not assign those scopes rather than fail
		claims, err := extractClaims(token.AccessToken)

		if err != nil {
			return ctx, fmt.Errorf("error extracting claims from token: %w", err)
		}

		for _, scope := range requiredScopes {
			if !claims.HasScope(scope) {
				return ctx, fmt.Errorf("authenticated successfully, but you don't have the required permission: '%v'", scope)
			}
		}

		log.WithContext(ctx).Info("Authenticated successfully ✅")

		// Save the token locally
		if home, err := os.UserHomeDir(); err == nil {
			// Create the directory if it doesn't exist
			err = os.MkdirAll(filepath.Join(home, ".overmind"), 0700)
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Failed to create ~/.overmind directory")
			}

			// Write the token to a file
			path := filepath.Join(home, ".overmind", "token.json")
			file, err := os.Create(path)
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to create token file at %v", path)
			}

			// Encode the token
			err = json.NewEncoder(file).Encode(token)
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("Failed to encode token file at %v", path)
			}

			log.WithContext(ctx).Debugf("Saved token to %v", path)
		}

		// Set the token
		return context.WithValue(ctx, sdp.UserTokenContextKey{}, token.AccessToken), nil
	}
	return ctx, fmt.Errorf("no OVM_API_KEY configured and target URL (%v) is insecure", parsed)
}

// getChangeUuid returns the UUID of a change, as selected by --uuid or --change, or a state with the specified status and having --ticket-link
func getChangeUuid(ctx context.Context, expectedStatus sdp.ChangeStatus, errNotFound bool) (uuid.UUID, error) {
	var changeUuid uuid.UUID
	var err error

	uuidString := viper.GetString("uuid")
	changeUrlString := viper.GetString("change")
	ticketLink := viper.GetString("ticket-link")

	// If no arguments are specified then return an error
	if uuidString == "" && changeUrlString == "" && ticketLink == "" {
		return uuid.Nil, errors.New("no change specified; use one of --change, --ticket-link or --uuid")
	}

	// Check UUID first if more than one is set
	if uuidString != "" {
		changeUuid, err = uuid.Parse(uuidString)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid --uuid value '%v', error: %w", uuidString, err)
		}

		return changeUuid, nil
	}

	// Then check for a change URL
	if changeUrlString != "" {
		return parseChangeUrl(changeUrlString)
	}

	// Finally look through all open changes to find one with a matching ticket link
	client := AuthenticatedChangesClient(ctx)

	var maybeChangeUuid *uuid.UUID
	changesList, err := client.ListChangesByStatus(ctx, &connect.Request[sdp.ListChangesByStatusRequest]{
		Msg: &sdp.ListChangesByStatusRequest{
			Status: expectedStatus,
		},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to searching for existing changes: %w", err)
	}

	for _, c := range changesList.Msg.GetChanges() {
		if c.GetProperties().GetTicketLink() == ticketLink {
			maybeChangeUuid = c.GetMetadata().GetUUIDParsed()
			if maybeChangeUuid != nil {
				changeUuid = *maybeChangeUuid
				break
			}
		}
	}

	if errNotFound && changeUuid == uuid.Nil {
		return uuid.Nil, fmt.Errorf("no change found with ticket link %v", ticketLink)
	}

	return changeUuid, nil
}

func parseChangeUrl(changeUrlString string) (uuid.UUID, error) {
	changeUrl, err := url.ParseRequestURI(changeUrlString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', error: %w", changeUrlString, err)
	}
	pathParts := strings.Split(path.Clean(changeUrl.Path), "/")
	if len(pathParts) < 2 {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', not long enough: %w", changeUrlString, err)
	}
	changeUuid, err := uuid.Parse(pathParts[2])
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid --change value '%v', couldn't parse UUID: %w", changeUrlString, err)
	}
	return changeUuid, nil
}

func withChangeUuidFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("change", "", "The frontend URL of the change to get")
	cmd.PersistentFlags().String("ticket-link", "", "Link to the ticket for this change.")
	cmd.PersistentFlags().String("uuid", "", "The UUID of the change that should be displayed.")
	cmd.MarkFlagsMutuallyExclusive("change", "ticket-link", "uuid")
}

func init() {
	cobra.OnInitialize(initConfig)

	// General Config
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")

	// api endpoint
	rootCmd.PersistentFlags().String("url", "https://api.prod.overmind.tech", "The overmind API endpoint")

	// Support API Keys in the environment
	err := viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY")
	if err != nil {
		log.WithError(err).Fatal("could not bind api key to env")
	}

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb. This requires --otel to be set.")
	// Mark this as hidden. This means that it will still be parsed of supplied,
	// and we will still look for it in the environment, but it won't be shown
	// in the help
	rootCmd.Flags().MarkHidden("honeycomb-api-key")

	// Create groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "iac",
		Title: "Infrastructure as Code:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "api",
		Title: "Overmind API:",
	})

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		formatter := new(log.TextFormatter)
		formatter.DisableTimestamp = true
		log.SetFormatter(formatter)

		// Read env vars
		var lvl log.Level

		if logLevel != "" {
			lvl, err = log.ParseLevel(logLevel)
			if err != nil {
				log.WithFields(log.Fields{"level": logLevel, "err": err}).Errorf("couldn't parse `log` config, defaulting to `info`")
				lvl = log.InfoLevel
			}
		} else {
			lvl = log.InfoLevel
		}
		log.SetLevel(lvl)

		if honeycombApiKey := viper.GetString("honeycomb-api-key"); honeycombApiKey != "" {
			if err := tracing.InitTracerWithHoneycomb(honeycombApiKey); err != nil {
				log.Fatal(err)
			}

			log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
				log.AllLevels[:log.GetLevel()+1]...,
			)))

			// shut down tracing at the end of the process
			rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
				tracing.ShutdownTracer()
			}
		}
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	replacer := strings.NewReplacer("-", "_")

	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match
}
