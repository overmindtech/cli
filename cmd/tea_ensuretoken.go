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
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/overmindtech/sdp-go"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
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
	deviceConfig  oauth2.Config
	deviceCode    *oauth2.DeviceAuthResponse
	deviceError   error

	width int
}

func NewEnsureTokenModel(ctx context.Context, app string, apiKey string, requiredScopes []string, width int) tea.Model {
	return ensureTokenModel{
		ctx:            ctx,
		app:            app,
		apiKey:         apiKey,
		requiredScopes: requiredScopes,

		taskModel: NewTaskModel("Ensuring Token", width),

		errors: []string{},

		width: width,
	}
}

func (m ensureTokenModel) TaskModel() taskModel {
	return m.taskModel
}

func (m ensureTokenModel) Init() tea.Cmd {
	return m.taskModel.Init()
}

func (m ensureTokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case instanceLoadedMsg:
		m.oi = msg.instance
		m.status = taskStatusRunning
		cmds = append(cmds, m.ensureTokenCmd(m.ctx))
		if os.Getenv("CI") == "" {
			cmds = append(cmds, m.spinner.Tick)
		}
	case displayAuthorizationInstructionsMsg:
		m.deviceMessage = "manual"
		m.deviceConfig = msg.config
		m.deviceCode = msg.deviceCode
		m.deviceError = msg.err

		m.status = taskStatusDone // avoid console flickering to allow click to be registered
		m.title = "Manual device authorization."
		cmds = append(cmds, m.awaitTokenCmd)
	case waitingForAuthorizationMsg:
		m.deviceMessage = "browser"
		m.deviceConfig = msg.config
		m.deviceCode = msg.deviceCode

		m.title = "Waiting for device authorization, check your browser."

		cmds = append(cmds, m.awaitTokenCmd)
	case tokenLoadedMsg:
		m.status = taskStatusDone
		m.title = "Using stored token"
		m.deviceMessage = ""
		cmds = append(cmds, m.tokenAvailable(msg.token))
	case tokenReceivedMsg:
		m.status = taskStatusDone
		m.title = "Authentication successful, using API key"
		m.deviceMessage = ""
		cmds = append(cmds, m.tokenAvailable(msg.token))
	case tokenStoredMsg:
		m.status = taskStatusDone
		m.title = fmt.Sprintf("Authentication successful, token stored locally (%v)", msg.file)
		m.deviceMessage = ""
		cmds = append(cmds, m.tokenAvailable(msg.token))
	case otherError:
		if msg.id == m.spinner.ID() {
			m.errors = append(m.errors, fmt.Sprintf("Note: %v", msg.err))
		}
	}

	var taskCmd tea.Cmd
	m.taskModel, taskCmd = m.taskModel.Update(msg)
	cmds = append(cmds, taskCmd)

	return m, tea.Batch(cmds...)
}

func (m ensureTokenModel) View() string {
	bits := []string{m.taskModel.View()}

	for _, err := range m.errors {
		bits = append(bits, wrap(fmt.Sprintf("  %v", err), m.width, 2))
	}
	switch m.deviceMessage {
	case "manual":
		beginAuthMessage := `# Authenticate with a browser

Automatically opening the SSO authorization page in your default browser failed: %v

Please open the following URL in your browser to authenticate:

%v

Then enter the code:

	%v
`
		bits = append(bits, markdownToString(m.width, fmt.Sprintf(beginAuthMessage, m.deviceError, m.deviceCode.VerificationURI, m.deviceCode.UserCode)))
	case "browser":
		beginAuthMessage := `# Authenticate with a browser

Attempting to automatically open the SSO authorization page in your default browser.
If the browser does not open or you wish to use a different device to authorize this request, open the following URL:

%v

Then enter the code:

	%v
`
		bits = append(bits, markdownToString(m.width, fmt.Sprintf(beginAuthMessage, m.deviceCode.VerificationURI, m.deviceCode.UserCode)))
	}
	return strings.Join(bits, "\n")
}

// ensureTokenCmd gets a token from the environment or from the user, and returns a
// context holding the token that can be used by sdp-go's helper functions to
// authenticate against the API
func (m ensureTokenModel) ensureTokenCmd(ctx context.Context) tea.Cmd {
	if viper.GetString("ovm-test-fake") != "" {
		return func() tea.Msg {
			return displayAuthorizationInstructionsMsg{
				deviceCode: &oauth2.DeviceAuthResponse{
					DeviceCode:              "test-device-code",
					VerificationURI:         "https://example.com/verify",
					VerificationURIComplete: "https://example.com/verify-complete",
				},
				err: errors.New("test error"),
			}
		}
	}

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

	if m.oi.CLIClientID == "" || m.oi.Auth0Domain == "" {
		return fatalError{id: m.spinner.ID(), err: errors.New("missing client id or auth0 domain")}
	}

	// If we need to get a new token, request the required scopes on top of
	// whatever ones the current local, valid token has so that we don't
	// keep replacing it
	requestScopes := append(m.requiredScopes, localScopes...)

	// Authenticate using the oauth device authorization flow
	config := oauth2.Config{
		ClientID: m.oi.CLIClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%v/authorize", m.oi.Auth0Domain),
			TokenURL:      fmt.Sprintf("https://%v/oauth/token", m.oi.Auth0Domain),
			DeviceAuthURL: fmt.Sprintf("https://%v/oauth/device/code", m.oi.Auth0Domain),
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
	ctx := m.ctx
	if m.deviceCode == nil {
		return fatalError{id: m.spinner.ID(), err: errors.New("device code is nil")}
	}

	if viper.GetString("ovm-test-fake") != "" {
		time.Sleep(500 * time.Millisecond)
		token := oauth2.Token{
			AccessToken:  "fake access token",
			TokenType:    "fake",
			RefreshToken: "fake refresh token",
			Expiry:       time.Now().Add(1 * time.Hour),
		}
		path := "fake token file path"
		return tokenStoredMsg{tokenReceivedMsg: tokenReceivedMsg{&token}, file: path}
	}

	// if there is an actual expiry, limit the entire process to that time
	if !m.deviceCode.Expiry.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, m.deviceCode.Expiry)
		defer cancel()
	}

	// while the RFC requires the oauth2 library to use 5 as the default, Auth0
	// should be able to handle more. Hence we re-implement the
	m.deviceCode.Interval = 1

	var token *oauth2.Token
	var err error
	for {
		log.Trace("attempting to get token from auth0")
		// reset the deviceCode's expiry to at most 1.5 seconds
		m.deviceCode.Expiry = time.Now().Add(1500 * time.Millisecond)

		token, err = m.deviceConfig.DeviceAccessToken(ctx, m.deviceCode)
		if err == nil {
			// we got a token, continue below. kthxbye
			log.Trace("we got a token from auth0")
			break
		}

		// See https://github.com/golang/oauth2/issues/635,
		// https://github.com/golang/oauth2/pull/636,
		// https://go-review.googlesource.com/c/oauth2/+/476316
		if errors.Is(err, context.DeadlineExceeded) || strings.HasSuffix(err.Error(), "context deadline exceeded") {
			// the context has expired, we need to retry
			log.WithError(err).Trace("context.DeadlineExceeded - waiting for a second")
			time.Sleep(time.Second)
			continue
		}

		// re-implement DeviceAccessToken's logic, but faster
		e, isRetrieveError := err.(*oauth2.RetrieveError) // nolint:errorlint // we depend on DeviceAccessToken() returning an non-wrapped error
		if !isRetrieveError {
			log.WithError(err).Trace("error authorizing token")
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error authorizing token: %w", err)}
		}

		switch e.ErrorCode {
		case "slow_down":
			// // https://datatracker.ietf.org/doc/html/rfc8628#section-3.5
			// // "the interval MUST be increased by 5 seconds for this and all subsequent requests"
			// interval += 5
			// ticker.Reset(time.Duration(interval) * time.Second)
		case "authorization_pending":
			// retry
		case "expired_token":
		default:
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error authorizing token (%v): %w", e.ErrorCode, err)}
		}
	}

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

	log.WithContext(ctx).Debugf("Saved token to %v", path)
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

	if m.token == nil {
		fmt.Println("Error running program: no token received")
		os.Exit(1)
	}
	tok, err := josejwt.ParseSigned(m.token.AccessToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		fmt.Println("Error running program: received invalid token:", err)
		os.Exit(1)
	}
	out := josejwt.Claims{}
	customClaims := sdp.CustomClaims{}
	err = tok.UnsafeClaimsWithoutVerification(&out, &customClaims)
	if err != nil {
		fmt.Println("Error running program: received unparsable token:", err)
		os.Exit(1)
	}

	if cmdSpan != nil {
		cmdSpan.SetAttributes(
			attribute.Bool("ovm.cli.authenticated", true),
			attribute.String("ovm.cli.accountName", customClaims.AccountName),
			attribute.String("ovm.cli.userId", out.Subject),
		)
	}

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
