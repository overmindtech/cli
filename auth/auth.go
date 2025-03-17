package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"connectrpc.com/connect"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpconnect"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const UserAgentVersion = "0.1"

// TokenClient Represents something that is capable of getting NATS JWT tokens
// for a given set of NKeys
type TokenClient interface {
	// Returns a NATS token that can be used to connect
	GetJWT() (string, error)

	// Uses the NKeys associated with the token to sign some binary data
	Sign([]byte) ([]byte, error)
}

// BasicTokenClient stores a static token and returns it when called, ignoring
// any provided NKeys or context since it already has the token and doesn't need
// to make any requests
type BasicTokenClient struct {
	staticToken string
	staticKeys  nkeys.KeyPair
}

// assert interface implementation
var _ TokenClient = (*BasicTokenClient)(nil)

// NewBasicTokenClient Creates a new basic token client that simply returns a static token
func NewBasicTokenClient(token string, keys nkeys.KeyPair) *BasicTokenClient {
	return &BasicTokenClient{
		staticToken: token,
		staticKeys:  keys,
	}
}

func (b *BasicTokenClient) GetJWT() (string, error) {
	return b.staticToken, nil
}

func (b *BasicTokenClient) Sign(in []byte) ([]byte, error) {
	return b.staticKeys.Sign(in)
}

// ClientCredentialsConfig Authenticates to Overmind using the Client
// Credentials flow
// https://auth0.com/docs/get-started/authentication-and-authorization-flow/client-credentials-flow
type ClientCredentialsConfig struct {
	// The ClientID of the application that we'll be authenticating as
	ClientID string
	// ClientSecret that corresponds to the ClientID
	ClientSecret string
}

type TokenSourceOptionsFunc func(*clientcredentials.Config)

// This option means that the token that is retrieved will have the following
// account embedded in it through impersonation. In order for this to work, the
// Auth0 ClientID must be added to workspace/deploy/auth0.tf. This will use
// deploy/auth0_embed_account_m2m.tftpl to update the Auth0 action that we use
// to allow impersonation. If this isn't done first you will get an error from
// Auth0.
func WithImpersonateAccount(account string) TokenSourceOptionsFunc {
	return func(c *clientcredentials.Config) {
		c.EndpointParams.Set("account_name", account)
	}
}

// TokenSource Returns a token source that can be used to get OAuth tokens.
// Cache this between invocations to avoid additional charges by Auth0 for M2M
// tokens. The oAuthTokenURL looks like this:
// https://somedomain.auth0.com/oauth/token
//
// The context that is passed to this function is used when getting new tokens,
// which will happen initially, and then subsequently when the token expires.
// This means that if this token source is going to be stored and used for many
// requests, it should not use the context of the request that created it, as
// this will be cancelled. Instead it should probably use `context.Background()`
// or similar.
func (flowConfig ClientCredentialsConfig) TokenSource(ctx context.Context, oAuthTokenURL, oAuthAudience string, opts ...TokenSourceOptionsFunc) oauth2.TokenSource {
	// inject otel into oauth2
	ctx = context.WithValue(ctx, oauth2.HTTPClient, otelhttp.DefaultClient)

	conf := &clientcredentials.Config{
		ClientID:     flowConfig.ClientID,
		ClientSecret: flowConfig.ClientSecret,
		TokenURL:     oAuthTokenURL,
		EndpointParams: url.Values{
			"audience": []string{oAuthAudience},
		},
	}

	for _, opt := range opts {
		opt(conf)
	}
	// this will be a `oauth2.ReuseTokenSource`, thus caching the M2M token.
	// note that this token source is safe for concurrent use and will
	// automatically refresh the token when it expires. Also note that this
	// token source will use the passed in http client from otelhttp for all
	// requests, but will not get the actual caller's context, so spans will not
	// link up.
	return conf.TokenSource(ctx)
}

// natsTokenClient A client that is capable of getting NATS JWTs and signing the
// required nonce to prove ownership of the NKeys. Satisfies the `TokenClient`
// interface
type natsTokenClient struct {
	// The name of the account to impersonate. If this is omitted then the
	// account will be determined based on the account included in the resulting
	// token.
	Account string

	// authenticated clients for the Overmind API
	adminClient sdpconnect.AdminServiceClient
	mgmtClient  sdpconnect.ManagementServiceClient

	jwt  string
	keys nkeys.KeyPair
}

// assert interface implementation
var _ TokenClient = (*natsTokenClient)(nil)

// generateKeys Generates a new set of keys for the client
func (n *natsTokenClient) generateKeys() error {
	var err error

	n.keys, err = nkeys.CreateUser()

	return err
}

// generateJWT Gets a new JWT from the auth API
func (n *natsTokenClient) generateJWT(ctx context.Context) error {
	if n.adminClient == nil || n.mgmtClient == nil {
		return errors.New("no Overmind API client configured")
	}

	// If we don't yet have keys generate them
	if n.keys == nil {
		err := n.generateKeys()

		if err != nil {
			return err
		}
	}

	pubKey, err := n.keys.PublicKey()
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	req := &sdp.CreateTokenRequest{
		UserPublicNkey: pubKey,
		UserName:       hostname,
	}

	// Create the request for a NATS token
	var response *connect.Response[sdp.CreateTokenResponse]
	if n.Account == "" {
		// Use the regular API and let the client authentication determine what our org should be
		log.WithFields(log.Fields{
			"account":    n.Account,
			"publicNKey": req.GetUserPublicNkey(),
			"UserName":   req.GetUserName(),
		}).Trace("Using regular API to get NATS token")
		response, err = n.mgmtClient.CreateToken(ctx, connect.NewRequest(req))
	} else {
		log.WithFields(log.Fields{
			"account":    n.Account,
			"publicNKey": req.GetUserPublicNkey(),
			"UserName":   req.GetUserName(),
		}).Trace("Using admin API to get NATS token")
		// Explicitly request an org
		response, err = n.adminClient.CreateToken(ctx, connect.NewRequest(&sdp.AdminCreateTokenRequest{
			Account: n.Account,
			Request: req,
		}))
	}
	if err != nil {
		return fmt.Errorf("getting NATS token failed: %w", err)
	}

	n.jwt = response.Msg.GetToken()

	return nil
}

func (n *natsTokenClient) GetJWT() (string, error) {
	ctx, span := tracer.Start(context.Background(), "connect.GetJWT")
	defer span.End()

	// If we don't yet have a JWT, generate one
	if n.jwt == "" {
		err := n.generateJWT(ctx)
		if err != nil {
			err = fmt.Errorf("error generating JWT: %w", err)
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}
	}

	claims, err := jwt.DecodeUserClaims(n.jwt)
	if err != nil {
		err = fmt.Errorf("error decoding JWT: %w", err)
		span.SetStatus(codes.Error, err.Error())
		return n.jwt, err
	}

	// Validate to make sure the JWT is valid. If it isn't we'll generate a new
	// one
	var vr jwt.ValidationResults

	claims.Validate(&vr)

	if vr.IsBlocking(true) {
		// Regenerate the token
		err := n.generateJWT(ctx)
		if err != nil {
			err = fmt.Errorf("error validating JWT: %w", err)
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}
	}

	span.SetStatus(codes.Ok, "Completed")
	return n.jwt, nil
}

func (n *natsTokenClient) Sign(in []byte) ([]byte, error) {
	if n.keys == nil {
		err := n.generateKeys()

		if err != nil {
			return []byte{}, err
		}
	}

	return n.keys.Sign(in)
}

// An OAuth2 token source which uses an Overmind API token as a source for OAuth
// tokens
type APIKeyTokenSource struct {
	// The API Key to use to authenticate to the Overmind API
	ApiKey       string
	token        *oauth2.Token
	apiKeyClient sdpconnect.ApiKeyServiceClient
}

func NewAPIKeyTokenSource(apiKey string, overmindAPIURL string) *APIKeyTokenSource {
	httpClient := http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Create a client that exchanges the API key for a JWT
	apiKeyClient := sdpconnect.NewApiKeyServiceClient(&httpClient, overmindAPIURL)

	return &APIKeyTokenSource{
		ApiKey:       apiKey,
		apiKeyClient: apiKeyClient,
	}
}

// Exchange an API key for an OAuth token
func (ats *APIKeyTokenSource) Token() (*oauth2.Token, error) {
	if ats.token != nil {
		// If we already have a token, and it is valid, return it
		if ats.token.Valid() {
			return ats.token, nil
		}
	}

	// Get a new token
	res, err := ats.apiKeyClient.ExchangeKeyForToken(context.Background(), connect.NewRequest(&sdp.ExchangeKeyForTokenRequest{
		ApiKey: ats.ApiKey,
	}))

	if err != nil {
		return nil, fmt.Errorf("error exchanging API key: %w", err)
	}

	if res.Msg.GetAccessToken() == "" {
		return nil, errors.New("no access token returned")
	}

	// Parse the expiry out of the token
	token, err := josejwt.ParseSigned(res.Msg.GetAccessToken(), []jose.SignatureAlgorithm{jose.RS256})

	if err != nil {
		return nil, fmt.Errorf("error parsing JWT: %w", err)
	}

	claims := josejwt.Claims{}

	err = token.UnsafeClaimsWithoutVerification(&claims)

	if err != nil {
		return nil, fmt.Errorf("error parsing JWT claims: %w", err)
	}

	ats.token = &oauth2.Token{
		AccessToken: res.Msg.GetAccessToken(),
		TokenType:   "Bearer",
		Expiry:      claims.Expiry.Time(),
	}

	return ats.token, nil
}

// NewAPIKeyClient Creates a new token client that authenticates to Overmind
// using an API key. This is exchanged for an OAuth token, which is then used to
// get a NATS token.
//
// The provided `overmindAPIURL` parameter should be the root URL of the
// Overmind API, without the /api suffix e.g. https://api.app.overmind.tech
func NewAPIKeyClient(overmindAPIURL string, apiKey string) (*natsTokenClient, error) {
	// Create a token source that exchanges the API key for an OAuth token
	tokenSource := NewAPIKeyTokenSource(apiKey, overmindAPIURL)
	transport := oauth2.Transport{
		Source: tokenSource,
		Base:   http.DefaultTransport,
	}
	httpClient := http.Client{
		Transport: otelhttp.NewTransport(&transport),
	}

	return &natsTokenClient{
		adminClient: sdpconnect.NewAdminServiceClient(&httpClient, overmindAPIURL),
		mgmtClient:  sdpconnect.NewManagementServiceClient(&httpClient, overmindAPIURL),
	}, nil
}

// NewStaticTokenClient Creates a new token client that uses a static token
// The user must pass the Overmind API URL to configure the client to connect
// to, the raw JWT OAuth access token, and the type of token. This is almost
// always "Bearer"
func NewStaticTokenClient(overmindAPIURL, token, tokenType string) (*natsTokenClient, error) {
	transport := oauth2.Transport{
		Source: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
			TokenType:   tokenType,
		}),
	}

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(&transport),
	}

	return &natsTokenClient{
		adminClient: sdpconnect.NewAdminServiceClient(&httpClient, overmindAPIURL),
		mgmtClient:  sdpconnect.NewManagementServiceClient(&httpClient, overmindAPIURL),
	}, nil
}

// NewOAuthTokenClient creates a token client that uses the provided TokenSource
// to get a NATS token. `overmindAPIURL` is the root URL of the NATS token
// exchange API that will be used e.g. https://api.server.test/v1
//
// Tokens will be minted under the specified account as long as the client has
// admin permissions, if not, the account that is attached to the client via
// Auth0 metadata will be used
func NewOAuthTokenClient(overmindAPIURL string, account string, ts oauth2.TokenSource) *natsTokenClient {
	return NewOAuthTokenClientWithContext(context.Background(), overmindAPIURL, account, ts)
}

// NewOAuthTokenClientWithContext creates a token client that uses the provided
// TokenSource to get a NATS token. `overmindAPIURL` is the root URL of the NATS
// token exchange API that will be used e.g. https://api.server.test/v1
//
// Tokens will be minted under the specified account as long as the client has
// admin permissions, if not, the account that is attached to the client via
// Auth0 metadata will be used
//
// The provided context is used for cancellation and to lookup the HTTP client
// used by oauth2. See the oauth2.HTTPClient variable.
//
// Provide an account name and an admin token to create a token client for a
// foreign account.
func NewOAuthTokenClientWithContext(ctx context.Context, overmindAPIURL string, account string, ts oauth2.TokenSource) *natsTokenClient {
	authenticatedClient := oauth2.NewClient(ctx, ts)

	// backwards compatibility: remove previously existing "/api" suffix from URL for connect
	apiUrl, err := url.Parse(overmindAPIURL)
	if err == nil {
		apiUrl.Path = ""
		overmindAPIURL = apiUrl.String()
	}

	return &natsTokenClient{
		Account:     account,
		adminClient: sdpconnect.NewAdminServiceClient(authenticatedClient, overmindAPIURL),
		mgmtClient:  sdpconnect.NewManagementServiceClient(authenticatedClient, overmindAPIURL),
	}
}
