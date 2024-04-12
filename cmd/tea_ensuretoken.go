package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

type tokenLoadedMsg struct{ token *oauth2.Token }
type displayAuthorizationInstructionsMsg struct {
	config     oauth2.Config
	deviceCode *oauth2.DeviceAuthResponse
	err        error
}
type waitingForAuthorizationMsg struct {
	config     oauth2.Config
	deviceCode *oauth2.DeviceAuthResponse
}
type tokenReceivedMsg struct {
	token *oauth2.Token
}
type tokenStoredMsg struct {
	tokenReceivedMsg
	file string
}
type tokenAvailableMsg struct {
	token *oauth2.Token
}

// this tea.Model uses the apiKey to request a fresh auth0 token from the
// api-server. If no apiKey is available it either loads the auth0 token from a
// config file, or drives an interactive device authorization flow to get a new
// token. Results are delivered as either a tokenAvailableMsg or a fatalError.
type ensureTokenModel struct {
	taskModel

	ctx            context.Context
	apiKey         string
	app            string
	oi             OvermindInstance
	requiredScopes []string

	errors []string

	deviceMessage string
	config        oauth2.Config
	deviceCode    *oauth2.DeviceAuthResponse
}

func NewEnsureTokenModel(ctx context.Context, app string, apiKey string, requiredScopes []string) tea.Model {
	return ensureTokenModel{
		ctx:            ctx,
		app:            app,
		apiKey:         apiKey,
		requiredScopes: requiredScopes,

		taskModel: NewTaskModel("Ensuring Token"),

		errors: []string{},
	}
}

func (m ensureTokenModel) Init() tea.Cmd {
	return m.taskModel.Init()
}

func (m ensureTokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case instanceLoadedMsg:
		m.oi = msg.instance
		m.status = taskStatusRunning
		return m, tea.Batch(
			m.ensureTokenCmd(m.ctx),
			m.spinner.Tick,
		)
	case displayAuthorizationInstructionsMsg:
		m.config = msg.config
		m.deviceCode = msg.deviceCode

		m.title = "Manual device authorization."
		beginAuthMessage := `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

	%v

Then enter the code:

	%v
`
		m.deviceMessage = markdownToString(fmt.Sprintf(beginAuthMessage, msg.deviceCode.VerificationURI, msg.deviceCode.UserCode))
		return m, m.awaitTokenCmd
	case waitingForAuthorizationMsg:
		m.config = msg.config
		m.deviceCode = msg.deviceCode

		m.title = "Waiting for device authorization, check your browser."
		beginAuthMessage := `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

	%v

Then enter the code:

	%v
`
		m.deviceMessage = markdownToString(fmt.Sprintf(beginAuthMessage, msg.deviceCode.VerificationURI, msg.deviceCode.UserCode))
		return m, m.awaitTokenCmd
	case tokenLoadedMsg:
		m.status = taskStatusDone
		m.title = "Using stored token"
		return m, m.tokenAvailable(msg.token)
	case tokenReceivedMsg:
		m.status = taskStatusDone
		m.title = "Authentication successful, using API key"
		return m, m.tokenAvailable(msg.token)
	case tokenStoredMsg:
		m.status = taskStatusDone
		m.title = fmt.Sprintf("Authentication successful, token stored locally (%v)", msg.file)
		return m, m.tokenAvailable(msg.token)
	case otherError:
		if msg.id == m.spinner.ID() {
			m.errors = append(m.errors, fmt.Sprintf("Note: %v", msg.err))
		}
		return m, nil
	case fatalError:
		if msg.id == m.spinner.ID() {
			m.status = taskStatusError
			m.title = fmt.Sprintf("Ensuring Token Error: %v", msg.err)
		}
		return m, nil
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		return m, taskCmd
	}
}

func (m ensureTokenModel) View() string {
	view := m.taskModel.View()
	if len(m.errors) > 0 {
		view += fmt.Sprintf("\n%v\n", strings.Join(m.errors, "\n"))
	}
	if m.deviceMessage != "" {
		view += fmt.Sprintf("\n%v\n", m.deviceMessage)
	}
	return view
}

// ensureTokenCmd gets a token from the environment or from the user, and returns a
// context holding the token that can be used by sdp-go's helper functions to
// authenticate against the API
func (m ensureTokenModel) ensureTokenCmd(ctx context.Context) tea.Cmd {
	if m.apiKey == "" {
		log.WithContext(ctx).Debug("getting token from Oauth")
		return m.oauthTokenCmd
	} else {
		log.WithContext(ctx).Debug("getting token from API key")
		return m.getAPIKeyTokenCmd
	}
}

// Gets a token from Oauth with the required scopes. This method will also cache
// that token locally for use later, and will use the cached token if possible
func (m ensureTokenModel) oauthTokenCmd() tea.Msg {
	var localScopes []string

	// Check for a locally saved token in ~/.overmind
	if home, err := os.UserHomeDir(); err == nil {
		var localToken *oauth2.Token

		localToken, localScopes, err = readLocalToken(home, m.requiredScopes)

		if err != nil {
			log.WithContext(m.ctx).Debugf("Error reading local token, ignoring: %v", err)
		} else {
			// If we already have the right scopes, return the token
			return tokenLoadedMsg{token: localToken}
		}
	}

	// If we need to get a new token, request the required scopes on top of
	// whatever ones the current local, valid token has so that we don't
	// keep replacing it
	requestScopes := append(m.requiredScopes, localScopes...)

	// Authenticate using the oauth device authorization flow
	config := oauth2.Config{
		ClientID: viper.GetString("cli-auth0-client-id"),
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", viper.GetString("cli-auth0-domain")),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", viper.GetString("cli-auth0-domain")),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", viper.GetString("cli-auth0-domain")),
		},
		Scopes: requestScopes,
	}

	deviceCode, err := config.DeviceAuth(m.ctx,
		oauth2.SetAuthURLParam("audience", m.oi.Audience),
		oauth2.AccessTypeOffline,
	)
	if err != nil {
		return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error getting device code: %w", err)}
	}

	err = browser.OpenURL(deviceCode.VerificationURIComplete)
	if err != nil {
		return displayAuthorizationInstructionsMsg{config, deviceCode, err}
	}
	return waitingForAuthorizationMsg{config, deviceCode}
}

func (m ensureTokenModel) awaitTokenCmd() tea.Msg {
	token, err := m.config.DeviceAccessToken(m.ctx, m.deviceCode)
	if err != nil {
		return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error authorizing token: %w", err)}
	}

	span := trace.SpanFromContext(m.ctx)
	span.SetAttributes(attribute.Bool("ovm.cli.authenticated", true))

	// Save the token locally
	home, err := os.UserHomeDir()
	if err != nil {
		return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to get home directory: %w", err)}
	}

	// Create the directory if it doesn't exist
	err = os.MkdirAll(filepath.Join(home, ".overmind"), 0700)
	if err != nil {
		return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to create ~/.overmind directory: %w", err)}
	}

	// Write the token to a file
	path := filepath.Join(home, ".overmind", "token.json")
	file, err := os.Create(path)
	if err != nil {
		return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to create token file at %v: %w", path, err)}
	}

	// Encode the token
	err = json.NewEncoder(file).Encode(token)
	if err != nil {
		return otherError{id: m.spinner.ID(), err: fmt.Errorf("failed to encode token file at %v: %w", path, err)}
	}

	log.WithContext(m.ctx).Debugf("Saved token to %v", path)
	return tokenStoredMsg{tokenReceivedMsg: tokenReceivedMsg{token}, file: path}
}

// Gets a token using an API key
func (m ensureTokenModel) getAPIKeyTokenCmd() tea.Msg {
	ctx := m.ctx

	var token *oauth2.Token

	if !strings.HasPrefix(m.apiKey, "ovm_api_") {
		return fatalError{id: m.spinner.ID(), err: errors.New("OVM_API_KEY does not match pattern 'ovm_api_*'")}
	}

	// exchange api token for JWT
	client := UnauthenticatedApiKeyClient(ctx, m.oi)
	resp, err := client.ExchangeKeyForToken(ctx, &connect.Request[sdp.ExchangeKeyForTokenRequest]{
		Msg: &sdp.ExchangeKeyForTokenRequest{
			ApiKey: m.apiKey,
		},
	})
	if err != nil {
		return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error authenticating the API token: %w", err)}
	}
	log.WithContext(ctx).Debug("successfully got a token from the API key")

	token = &oauth2.Token{
		AccessToken: resp.Msg.GetAccessToken(),
		TokenType:   "Bearer",
	}

	return tokenReceivedMsg{token}
}

func (m ensureTokenModel) tokenAvailable(token *oauth2.Token) tea.Cmd {
	return func() tea.Msg {
		return tokenAvailableMsg{token}
	}
}

/////////////////////////////
//  "legacy" non-tea code  //
/////////////////////////////

// Gets a token from Oauth with the required scopes. This method will also cache
// that token locally for use later, and will use the cached token if possible
func getOauthToken(ctx context.Context, oi OvermindInstance, requiredScopes []string) (*oauth2.Token, error) {
	var localScopes []string

	// Check for a locally saved token in ~/.overmind
	if home, err := os.UserHomeDir(); err == nil {
		var localToken *oauth2.Token

		localToken, localScopes, err = readLocalToken(home, requiredScopes)

		if err != nil {
			log.WithContext(ctx).Debugf("Error reading local token, ignoring: %v", err)
		} else {
			// If we already have the right scopes, return the token
			return localToken, nil
		}
	}

	// If we need to get a new token, request the required scopes on top of
	// whatever ones the current local, valid token has so that we don't
	// keep replacing it
	requestScopes := append(requiredScopes, localScopes...)

	// Authenticate using the oauth device authorization flow
	config := oauth2.Config{
		ClientID: viper.GetString("cli-auth0-client-id"),
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", viper.GetString("cli-auth0-domain")),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", viper.GetString("cli-auth0-domain")),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", viper.GetString("cli-auth0-domain")),
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

	m := authenticateModel{ctx: ctx, status: PromptUser, deviceCode: deviceCode, config: config}
	authenticateProgram := tea.NewProgram(m)

	result, err := authenticateProgram.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	m, ok := result.(authenticateModel)
	if !ok {
		fmt.Println("Error running program: result is not authenticateModel")
		os.Exit(1)
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Bool("ovm.cli.authenticated", true))

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
		err = json.NewEncoder(file).Encode(m.token)
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("Failed to encode token file at %v", path)
		}

		log.WithContext(ctx).Debugf("Saved token to %v", path)
	}

	return m.token, nil
}

// ensureToken gets a token from the environment or from the user, and returns a
// context holding the token tthat can be used by sdp-go's helper functions to
// authenticate against the API
func ensureToken(ctx context.Context, oi OvermindInstance, requiredScopes []string) (context.Context, *oauth2.Token, error) {
	var token *oauth2.Token
	var err error

	// get a token from the api key if present
	if apiKey := viper.GetString("api-key"); apiKey != "" {
		token, err = getAPIKeyToken(ctx, oi, apiKey, requiredScopes)
	} else {
		token, err = getOauthToken(ctx, oi, requiredScopes)
	}
	if err != nil {
		return ctx, nil, fmt.Errorf("error getting token: %w", err)
	}

	// Check that we actually got the claims we asked for. If you don't have
	// permission auth0 will just not assign those scopes rather than fail
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return ctx, nil, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return ctx, nil, fmt.Errorf("authenticated successfully, but you don't have the required permission: '%v'", missing)
	}

	// store the token for later use by sdp-go's auth client. Note that this
	// loses access to the RefreshToken and could be done better by using an
	// oauth2.TokenSource, but this would require more work on updating sdp-go
	// that is currently not scheduled
	ctx = context.WithValue(ctx, sdp.UserTokenContextKey{}, token.AccessToken)

	return ctx, token, nil
}

// Returns whether a set of claims has all of the required scopes. It also
// accounts for when a user has write access but required read access, they
// aren't the same but the user will have access anyway so this will pass
//
// Returns true if the token has the required scopes. Otherwise, false and the missing permission for displaying or logging
func HasScopesFlexible(token *oauth2.Token, requiredScopes []string) (bool, string, error) {
	if token == nil {
		return false, "", errors.New("HasScopesFlexible: token is nil")
	}

	claims, err := extractClaims(token.AccessToken)
	if err != nil {
		return false, "", fmt.Errorf("error extracting claims from token: %w", err)
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			// If they don't have the *exact* scope, check to see if they have
			// write access to the same service
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
func readLocalToken(homeDir string, requiredScopes []string) (*oauth2.Token, []string, error) {
	// Read in the token JSON file
	path := filepath.Join(homeDir, ".overmind", "token.json")

	token := new(oauth2.Token)

	// Check that the file exists
	if _, err := os.Stat(path); err != nil {
		return nil, nil, err
	}

	// Read the file
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening token file at %v: %w", path, err)
	}

	// Decode the file
	err = json.NewDecoder(file).Decode(token)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding token file at %v: %w", path, err)
	}

	// Check to see if the token is still valid
	if !token.Valid() {
		return nil, nil, errors.New("token is no longer valid")
	}

	claims, err := extractClaims(token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("error extracting claims from token: %w", err)
	}
	if claims.Scope == "" {
		return nil, nil, errors.New("token does not have any scopes")
	}

	currentScopes := strings.Split(claims.Scope, " ")

	// Check that we actually got the claims we asked for.
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return nil, currentScopes, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return nil, currentScopes, fmt.Errorf("local token is missing this permission: '%v'", missing)
	}

	log.Debugf("Using local token from %v", path)
	return token, currentScopes, nil
}

// Gets a token using an API key
func getAPIKeyToken(ctx context.Context, oi OvermindInstance, apiKey string, requiredScopes []string) (*oauth2.Token, error) {
	log.WithContext(ctx).Debug("using provided token for authentication")

	var token *oauth2.Token

	if !strings.HasPrefix(apiKey, "ovm_api_") {
		return nil, errors.New("OVM_API_KEY does not match pattern 'ovm_api_*'")
	}

	// exchange api token for JWT
	client := UnauthenticatedApiKeyClient(ctx, oi)
	resp, err := client.ExchangeKeyForToken(ctx, &connect.Request[sdp.ExchangeKeyForTokenRequest]{
		Msg: &sdp.ExchangeKeyForTokenRequest{
			ApiKey: apiKey,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error authenticating the API token: %w", err)
	}
	log.WithContext(ctx).Debug("successfully got a token from the API key")

	token = &oauth2.Token{
		AccessToken: resp.Msg.GetAccessToken(),
		TokenType:   "Bearer",
	}

	// Check that we actually got the claims we asked for. If you don't have
	// permission auth0 will just not assign those scopes rather than fail
	ok, missing, err := HasScopesFlexible(token, requiredScopes)
	if err != nil {
		return nil, fmt.Errorf("error checking token scopes: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("authenticated successfully, but your API key is missing this permission: '%v'", missing)
	}

	return token, nil
}
