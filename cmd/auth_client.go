package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpconnect"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// UnauthenticatedApiKeyClient Returns an apikey client with otel instrumentation
// but no authentication. Can only be used for ExchangeKeyForToken
func UnauthenticatedApiKeyClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.ApiKeyServiceClient {
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind apikeys API")
	return sdpconnect.NewApiKeyServiceClient(otelhttp.DefaultClient, oi.ApiUrl.String())
}

// AuthenticatedBookmarkClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedBookmarkClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.BookmarksServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind bookmark API")
	return sdpconnect.NewBookmarksServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedChangesClient Returns a changes client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedChangesClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.ChangesServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind changes API")
	return sdpconnect.NewChangesServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedConfigurationClient  Returns a config client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedConfigurationClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.ConfigurationServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind configuration API")
	return sdpconnect.NewConfigurationServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedManagementClient Returns a management client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedManagementClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.ManagementServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind management API")
	return sdpconnect.NewManagementServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedSnapshotsClient Returns a Snapshots client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedSnapshotsClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.SnapshotsServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind snapshot API")
	return sdpconnect.NewSnapshotsServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedInviteClient Returns a Invite client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedInviteClient(ctx context.Context, oi sdp.OvermindInstance) sdpconnect.InviteServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	log.WithContext(ctx).WithField("apiUrl", oi.ApiUrl).Debug("Connecting to overmind invite API")
	return sdpconnect.NewInviteServiceClient(httpClient, oi.ApiUrl.String())
}

// AuthenticatedClient is a http.Client that will automatically add the required
// Authorization header to the request, which is taken from the context that it
// is created with. We also always set the X-overmind-interactive header to
// false
type AuthenticatedTransport struct {
	from http.RoundTripper
	ctx  context.Context
}

// NewAuthenticatedClient creates a new AuthenticatedClient from the given
// context and http.Client.
func NewAuthenticatedClient(ctx context.Context, from *http.Client) *http.Client {
	return &http.Client{
		Transport: &AuthenticatedTransport{
			from: from.Transport,
			ctx:  ctx,
		},
		CheckRedirect: from.CheckRedirect,
		Jar:           from.Jar,
		Timeout:       from.Timeout,
	}
}

// RoundTrip Adds the Authorization header to the request then call the
// underlying roundTripper
func (y *AuthenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// ask for otel trace linkup
	req.Header.Set("X-Overmind-Interactive", "false")

	// Extract auth from the context
	ctxToken := y.ctx.Value(auth.UserTokenContextKey{})

	if ctxToken != nil {
		token, ok := ctxToken.(string)

		if ok && token != "" {
			bearer := fmt.Sprintf("Bearer %v", token)
			req.Header.Set("Authorization", bearer)
		}
	}

	return y.from.RoundTrip(req)
}
