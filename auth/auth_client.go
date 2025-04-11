package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/overmindtech/cli/sdp-go/sdpconnect"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

// AuthenticatedAdminClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedAdminClient(ctx context.Context, apiUrl string) sdpconnect.AdminServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind admin API (pre-authenticated)")
	return sdpconnect.NewAdminServiceClient(httpClient, apiUrl)
}

// AuthenticatedApiKeyClient Returns an apikey client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedApiKeyClient(ctx context.Context, apiUrl string) sdpconnect.ApiKeyServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind apikeys API (pre-authenticated)")
	return sdpconnect.NewApiKeyServiceClient(httpClient, apiUrl)
}

// UnauthenticatedApiKeyClient Returns an apikey client with otel instrumentation
// but no authentication. Can only be used for ExchangeKeyForToken
func UnauthenticatedApiKeyClient(ctx context.Context, apiUrl string) sdpconnect.ApiKeyServiceClient {
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind apikeys API")
	return sdpconnect.NewApiKeyServiceClient(otelhttp.DefaultClient, apiUrl)
}

// AuthenticatedBookmarkClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedBookmarkClient(ctx context.Context, apiUrl string) sdpconnect.BookmarksServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind bookmark API (pre-authenticated)")
	return sdpconnect.NewBookmarksServiceClient(httpClient, apiUrl)
}

// AuthenticatedChangesClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedChangesClient(ctx context.Context, apiUrl string) sdpconnect.ChangesServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind changes API (pre-authenticated)")
	return sdpconnect.NewChangesServiceClient(httpClient, apiUrl)
}

// AuthenticatedConfigurationClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedConfigurationClient(ctx context.Context, apiUrl string) sdpconnect.ConfigurationServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind configuration API (pre-authenticated)")
	return sdpconnect.NewConfigurationServiceClient(httpClient, apiUrl)
}

// AuthenticatedManagementClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedManagementClient(ctx context.Context, apiUrl string) sdpconnect.ManagementServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind management API (pre-authenticated)")
	return sdpconnect.NewManagementServiceClient(httpClient, apiUrl)
}

// AuthenticatedSnapshotsClient Returns a Snapshots client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedSnapshotsClient(ctx context.Context, apiUrl string) sdpconnect.SnapshotsServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind snapshot API (pre-authenticated)")
	return sdpconnect.NewSnapshotsServiceClient(httpClient, apiUrl)
}

// AuthenticatedInviteClient Returns a Invite client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedInviteClient(ctx context.Context, apiUrl string) sdpconnect.InviteServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", apiUrl).Debug("Connecting to overmind invite API (pre-authenticated)")
	return sdpconnect.NewInviteServiceClient(httpClient, apiUrl)
}
