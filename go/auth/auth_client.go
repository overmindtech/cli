package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
)

// AuthenticatedClient is a http.Client that will automatically add the required
// Authorization header to the request, which is taken from the context that it
// is created with. We also always set the X-overmind-interactive header to
// false to connect opentelemetry traces.
type AuthenticatedTransport struct {
	from  http.RoundTripper
	token string
}

// RoundTrip Adds the Authorization header to the request then call the
// underlying roundTripper
func (y *AuthenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// ask for otel trace linkup
	req.Header.Set("X-Overmind-Interactive", "false")

	if y.token != "" {
		bearer := fmt.Sprintf("Bearer %v", y.token)
		req.Header.Set("Authorization", bearer)
	}

	return y.from.RoundTrip(req)
}

// NewAuthenticatedClient creates a new AuthenticatedClient from the given
// context and http.Client.
func NewAuthenticatedClient(ctx context.Context, from *http.Client) *http.Client {
	token, ok := ctx.Value(UserTokenContextKey{}).(string)
	if !ok {
		token = ""
	}

	return &http.Client{
		Transport: &AuthenticatedTransport{
			from:  from.Transport,
			token: token,
		},
		CheckRedirect: from.CheckRedirect,
		Jar:           from.Jar,
		Timeout:       from.Timeout,
	}
}

// ContextAwareAuthTransport is an http.RoundTripper that extracts the user JWT
// from each request's context at call time (not at client-creation time). This
// enables a single persistent http.Client to pass through per-request JWTs,
// which is needed when the client is created once at startup but serves
// requests from different users.
type ContextAwareAuthTransport struct {
	from http.RoundTripper
}

// RoundTrip extracts the JWT from the request's context and adds it as a
// Bearer token in the Authorization header.
func (t *ContextAwareAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Overmind-Interactive", "false")

	if token, ok := req.Context().Value(UserTokenContextKey{}).(string); ok && token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	}

	return t.from.RoundTrip(req)
}

// NewContextAwareAuthClient creates an http.Client whose transport extracts the
// JWT from each outgoing request's context. Unlike NewAuthenticatedClient (which
// captures the token once), this client re-reads the token on every call —
// making it safe to reuse across requests from different users.
func NewContextAwareAuthClient(from *http.Client) *http.Client {
	return &http.Client{
		Transport: &ContextAwareAuthTransport{
			from: from.Transport,
		},
		CheckRedirect: from.CheckRedirect,
		Jar:           from.Jar,
		Timeout:       from.Timeout,
	}
}

// AuthenticatedAdminClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedAdminClient(ctx context.Context, apiUrl string) sdpconnect.AdminServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind admin API (pre-authenticated)")
	return sdpconnect.NewAdminServiceClient(httpClient, apiUrl)
}

// AuthenticatedApiKeyClient Returns an apikey client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedApiKeyClient(ctx context.Context, apiUrl string) sdpconnect.ApiKeyServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind apikeys API (pre-authenticated)")
	return sdpconnect.NewApiKeyServiceClient(httpClient, apiUrl)
}

// UnauthenticatedApiKeyClient Returns an apikey client with otel instrumentation
// but no authentication. Can only be used for ExchangeKeyForToken
func UnauthenticatedApiKeyClient(ctx context.Context, apiUrl string) sdpconnect.ApiKeyServiceClient {
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind apikeys API")
	return sdpconnect.NewApiKeyServiceClient(tracing.HTTPClient(), apiUrl)
}

// AuthenticatedBookmarkClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedBookmarkClient(ctx context.Context, apiUrl string) sdpconnect.BookmarksServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind bookmark API (pre-authenticated)")
	return sdpconnect.NewBookmarksServiceClient(httpClient, apiUrl)
}

// AuthenticatedChangesClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedChangesClient(ctx context.Context, apiUrl string) sdpconnect.ChangesServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind changes API (pre-authenticated)")
	return sdpconnect.NewChangesServiceClient(httpClient, apiUrl)
}

// AuthenticatedConfigurationClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedConfigurationClient(ctx context.Context, apiUrl string) sdpconnect.ConfigurationServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind configuration API (pre-authenticated)")
	return sdpconnect.NewConfigurationServiceClient(httpClient, apiUrl)
}

// AuthenticatedManagementClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedManagementClient(ctx context.Context, apiUrl string) sdpconnect.ManagementServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind management API (pre-authenticated)")
	return sdpconnect.NewManagementServiceClient(httpClient, apiUrl)
}

// AuthenticatedSnapshotsClient Returns a Snapshots client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedSnapshotsClient(ctx context.Context, apiUrl string) sdpconnect.SnapshotsServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind snapshot API (pre-authenticated)")
	return sdpconnect.NewSnapshotsServiceClient(httpClient, apiUrl)
}

// AuthenticatedInviteClient Returns a Invite client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedInviteClient(ctx context.Context, apiUrl string) sdpconnect.InviteServiceClient {
	httpClient := NewAuthenticatedClient(ctx, tracing.HTTPClient())
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind invite API (pre-authenticated)")
	return sdpconnect.NewInviteServiceClient(httpClient, apiUrl)
}
