package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpconnect"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// AuthenticatedBookmarkClient Returns a bookmark client that uses the auth
// embedded in the context and otel instrumentation
func AuthenticatedChangesClient(ctx context.Context) sdpconnect.ChangesServiceClient {
	httpClient := NewAuthenticatedClient(ctx, otelhttp.DefaultClient)
	url := viper.GetString("changes-url")
	if url == "" {
		url = viper.GetString("url")
	}
	return sdpconnect.NewChangesServiceClient(httpClient, url)
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
	req.Header.Set("X-overmind-interactive", "false")

	// Extract auth from the context
	ctxToken := y.ctx.Value(sdp.UserTokenContextKey{})

	if ctxToken != nil {
		token, ok := ctxToken.(string)

		if ok && token != "" {
			bearer := fmt.Sprintf("Bearer %v", token)
			req.Header.Set("Authorization", bearer)
		}
	}

	return y.from.RoundTrip(req)
}
