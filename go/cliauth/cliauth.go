// Package cliauth provides shared CLI authentication logic for OAuth device flow,
// API key exchange, and token caching.
//
// This package is used by both the public overmind CLI and the area51-cli to avoid
// code duplication and ensure consistent authentication behavior.
package cliauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/go/auth"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/go/tracing"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

// Logger is an interface for outputting authentication messages.
// Implementations can use pterm, slog, or any other logging framework.
type Logger interface {
	Info(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// TokenFile represents the ~/.overmind/token.json file structure.
// This format is shared between all Overmind CLI tools.
type TokenFile struct {
	AuthEntries map[string]*TokenEntry `json:"auth_entries"`
}

// TokenEntry represents a single auth entry in the token file
type TokenEntry struct {
	Token     *oauth2.Token `json:"token"`
	AddedDate time.Time     `json:"added_date"`
}

// ReadLocalToken reads a cached token from ~/.overmind/token.json for the given
// app URL. Returns the token and its current scopes if valid and sufficient.
func ReadLocalToken(homeDir, app string, requiredScopes []string, log Logger) (*oauth2.Token, []string, error) {
	path := filepath.Join(homeDir, ".overmind", "token.json")

	tokenFile := new(TokenFile)

	if _, err := os.Stat(path); err != nil {
		return nil, nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening token file at %q: %w", path, err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(tokenFile)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding token file at %q: %w", path, err)
	}

	authEntry, ok := tokenFile.AuthEntries[app]
	if !ok {
		return nil, nil, fmt.Errorf("no token found for app %s in %q", app, path)
	}

	if authEntry == nil {
		return nil, nil, fmt.Errorf("token entry for app %s is null in %q", app, path)
	}

	if authEntry.Token == nil {
		return nil, nil, fmt.Errorf("token for app %s is null in %q", app, path)
	}
	if !authEntry.Token.Valid() {
		return nil, nil, errors.New("token is no longer valid")
	}

	claims, err := ExtractClaims(authEntry.Token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting claims from token: %s in %q: %w", app, path, err)
	}
	if claims.Scope == "" {
		return nil, nil, errors.New("token does not have any scopes")
	}

	currentScopes := strings.Split(claims.Scope, " ")

	ok, missing, err := HasScopesFlexible(authEntry.Token, requiredScopes)
	if err != nil {
		return nil, currentScopes, fmt.Errorf("error checking token scopes: %s in %q: %w", app, path, err)
	}
	if !ok {
		return nil, currentScopes, fmt.Errorf("local token is missing this permission: '%v'. %s in %q", missing, app, path)
	}

	log.Info("Using local token", "app", app, "path", path)
	return authEntry.Token, currentScopes, nil
}

// SaveLocalToken saves a token to ~/.overmind/token.json with secure permissions
// (directory 0700, file 0600). The token is keyed by app URL so multiple
// environments can coexist.
func SaveLocalToken(homeDir, app string, token *oauth2.Token, log Logger) error {
	path := filepath.Join(homeDir, ".overmind", "token.json")
	dir := filepath.Dir(path)

	tokenFile := &TokenFile{
		AuthEntries: make(map[string]*TokenEntry),
	}

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()

			err = json.NewDecoder(file).Decode(tokenFile)
			if err != nil {
				return fmt.Errorf("error decoding token file at %q: %w", path, err)
			}

			if tokenFile.AuthEntries == nil {
				tokenFile.AuthEntries = make(map[string]*TokenEntry)
			}
		}
	} else {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return fmt.Errorf("unexpected fail creating directories: %w", err)
		}
	}

	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("failed to set directory permissions: %w", err)
	}

	tokenFile.AuthEntries[app] = &TokenEntry{
		Token:     token,
		AddedDate: time.Now(),
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error creating token file at %q: %w", path, err)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(tokenFile)
	if err != nil {
		return fmt.Errorf("error encoding token file at %q: %w", path, err)
	}

	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	log.Info("Saved token locally", "app", app, "path", path)
	return nil
}

// HasScopesFlexible checks if a token has the required scopes. A service:write
// scope is treated as satisfying service:read.
func HasScopesFlexible(token *oauth2.Token, requiredScopes []string) (bool, string, error) {
	if token == nil {
		return false, "", errors.New("HasScopesFlexible: token is nil")
	}

	claims, err := ExtractClaims(token.AccessToken)
	if err != nil {
		return false, "", fmt.Errorf("error extracting claims from token: %w", err)
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			sections := strings.Split(scope, ":")
			var hasWriteInstead bool

			if len(sections) == 2 {
				service, action := sections[0], sections[1]
				if action == "read" {
					hasWriteInstead = claims.HasScope(fmt.Sprintf("%v:write", service))
				}
			}

			if !hasWriteInstead {
				return false, scope, nil
			}
		}
	}

	return true, "", nil
}

// ExtractClaims extracts custom claims from a JWT token without verifying the
// signature. Signature verification is the server's responsibility; we only
// need the claims for scope checking.
func ExtractClaims(token string) (*auth.CustomClaims, error) {
	sections := strings.Split(token, ".")
	if len(sections) != 3 {
		return nil, errors.New("token is not a JWT")
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(sections[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding token payload: %w", err)
	}

	claims := new(auth.CustomClaims)
	err = json.Unmarshal(decodedPayload, claims)
	if err != nil {
		return nil, fmt.Errorf("error parsing token payload: %w", err)
	}

	return claims, nil
}

// GetOauthToken authenticates using the OAuth2 device authorization flow.
// It first checks for a cached token in ~/.overmind/token.json and falls back
// to the interactive device flow if needed. New tokens are cached for reuse.
func GetOauthToken(ctx context.Context, oi sdp.OvermindInstance, app string, requiredScopes []string, log Logger) (*oauth2.Token, error) {
	var localScopes []string
	var localToken *oauth2.Token
	home, err := os.UserHomeDir()
	if err == nil {
		localToken, localScopes, err = ReadLocalToken(home, app, requiredScopes, log)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Info("Skipping local token, re-authenticating", "error", err.Error())
			}
		} else {
			return localToken, nil
		}
	}

	// Request the required scopes on top of whatever the current local token
	// has so that we don't keep replacing it with one that has fewer scopes.
	// Use a new slice to avoid mutating the caller's requiredScopes.
	requestScopes := make([]string, 0, len(requiredScopes)+len(localScopes))
	requestScopes = append(requestScopes, requiredScopes...)
	requestScopes = append(requestScopes, localScopes...)

	config := oauth2.Config{
		ClientID: oi.CLIClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", oi.Auth0Domain),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", oi.Auth0Domain),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", oi.Auth0Domain),
		},
		Scopes: requestScopes,
	}

	deviceCode, err := config.DeviceAuth(ctx,
		oauth2.SetAuthURLParam("audience", oi.Audience),
		oauth2.AccessTypeOffline,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting device code: %w", err)
	}

	var urlToOpen string
	if deviceCode.VerificationURIComplete != "" {
		urlToOpen = deviceCode.VerificationURIComplete
	} else {
		urlToOpen = deviceCode.VerificationURI
	}

	_ = browser.OpenURL(urlToOpen)
	log.Info("Open this URL in your browser to authenticate",
		"url", deviceCode.VerificationURI,
		"code", deviceCode.UserCode)

	token, err := config.DeviceAccessToken(ctx, deviceCode)
	if err != nil {
		log.Error("Unable to authenticate. Please try again.", "error", err.Error())
		return nil, fmt.Errorf("error getting device access token: %w", err)
	}
	if token == nil {
		log.Error("No token received")
		return nil, errors.New("no token received")
	}

	log.Info("Authenticated successfully")

	if home != "" {
		err = SaveLocalToken(home, app, token, log)
		if err != nil {
			log.Error("Error saving token", "error", err.Error())
		}
	}

	return token, nil
}

// GetAPIKeyToken exchanges an Overmind API key (ovm_api_*) for a JWT token
// via the ApiKeyService, then verifies the token has the required scopes.
func GetAPIKeyToken(ctx context.Context, oi sdp.OvermindInstance, app, apiKey string, requiredScopes []string, log Logger) (*oauth2.Token, error) {
	if !strings.HasPrefix(apiKey, "ovm_api_") {
		return nil, errors.New("API key does not match pattern 'ovm_api_*'")
	}

	httpClient := tracing.HTTPClient()
	client := sdpconnect.NewApiKeyServiceClient(httpClient, oi.ApiUrl.String())

	resp, err := client.ExchangeKeyForToken(ctx, &connect.Request[sdp.ExchangeKeyForTokenRequest]{
		Msg: &sdp.ExchangeKeyForTokenRequest{
			ApiKey: apiKey,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error authenticating the API token for %s: %w", app, err)
	}

	token := &oauth2.Token{
		AccessToken: resp.Msg.GetAccessToken(),
		TokenType:   "Bearer",
	}

	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return nil, fmt.Errorf("error checking token scopes for %s: %w", app, err)
	}
	if !ok {
		return nil, fmt.Errorf("authenticated successfully against %s, but your API key is missing this permission: '%v'", app, missing)
	}
	log.Info("Using Overmind API key", "app", app)
	return token, nil
}

// GetToken gets a token using either API key exchange (if apiKey is non-empty)
// or the OAuth device flow.
func GetToken(ctx context.Context, oi sdp.OvermindInstance, app, apiKey string, requiredScopes []string, log Logger) (*oauth2.Token, error) {
	if apiKey != "" {
		return GetAPIKeyToken(ctx, oi, app, apiKey, requiredScopes, log)
	}
	return GetOauthToken(ctx, oi, app, requiredScopes, log)
}
