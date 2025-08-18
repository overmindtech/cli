package auth

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nkeys"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpconnect"
)

var tokenExchangeURLs = []string{
	"http://api-server:8080",
	"http://localhost:8080",
}

func TestBasicTokenClient(t *testing.T) {
	var c TokenClient

	keys, err := nkeys.CreateUser()

	if err != nil {
		t.Fatal(err)
	}

	c = NewBasicTokenClient("tokeny_mc_tokenface", keys)

	var token string

	token, err = c.GetJWT()

	if err != nil {
		t.Error(err)
	}

	if token != "tokeny_mc_tokenface" {
		t.Error("token mismatch")
	}

	data := []byte{1, 156, 230, 4, 23, 175, 11}

	signed, err := c.Sign(data)

	if err != nil {
		t.Fatal(err)
	}

	err = keys.Verify(data, signed)

	if err != nil {
		t.Error(err)
	}
}

func GetTestOAuthTokenClient(t *testing.T) *natsTokenClient {
	var domain string
	var clientID string
	var clientSecret string
	var exists bool

	errorFormat := "environment variable %v not found. Set up your test environment first. See: https://github.com/overmindtech/cli/auth0-test-data"

	// Read secrets form the environment
	if domain, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_DOMAIN"); !exists || domain == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_DOMAIN")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientID, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_ID"); !exists || clientID == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_ID")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientSecret, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_SECRET"); !exists || clientSecret == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_SECRET")
		t.Skip("Skipping due to missing environment setup")
	}

	exchangeURL, err := GetWorkingTokenExchange()

	if err != nil {
		t.Fatal(err)
	}

	flowConfig := ClientCredentialsConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	return NewOAuthTokenClient(
		exchangeURL,
		"overmind-development",
		flowConfig.TokenSource(t.Context(), fmt.Sprintf("https://%v/oauth/token", domain), os.Getenv("API_SERVER_AUDIENCE")),
	)
}

func TestOAuthTokenClient(t *testing.T) {
	c := GetTestOAuthTokenClient(t)

	var err error

	_, err = c.GetJWT()

	if err != nil {
		t.Error(err)
	}

	// Make sure it can sign
	data := []byte{1, 156, 230, 4, 23, 175, 11}

	_, err = c.Sign(data)

	if err != nil {
		t.Fatal(err)
	}

}

type testAPIKeyHandler struct {
	sdpconnect.UnimplementedApiKeyServiceHandler
}

// Always return a valid token
func (h *testAPIKeyHandler) ExchangeKeyForToken(ctx context.Context, req *connect.Request[sdp.ExchangeKeyForTokenRequest]) (*connect.Response[sdp.ExchangeKeyForTokenResponse], error) {
	return &connect.Response[sdp.ExchangeKeyForTokenResponse]{
		Msg: &sdp.ExchangeKeyForTokenResponse{
			AccessToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMiwiZXhwIjoxNTE2MjM5MDIyfQ.Tt0D8zOO3uzfbR1VLc3v7S1_jNrP9_crU1Gi_LpVinEXn4hndTWnI9rMd9r9D0iiv6U-CZAb9JKlun58MO3Pbf_S7apiLGHGE11coIMdk5OKuQFepwXPEk4ixs8_51wmWtJAKg7L5JJG6NuLGnGK8a53hzSHjoK80ROBqlsE9dJ4lpgigj8ZcL-xWpjS4TnUiGLHOvNDnHdqP5D_3DA1teWk9PNh9uU6Wn3U3ShH9rRCI9mKz9amdZ7QzH44J5Gsh2-uo0m2BtZILBE5_p-BeJ7op2RicEXbm69Vae8SPjkJLorBQxbO2lMG4y00q1n-wRDfg_eLFH8ZVC-5lpVXIw",
		},
	}, nil
}

func TestNewAPIKeyTokenSource(t *testing.T) {
	_, handler := sdpconnect.NewApiKeyServiceHandler(&testAPIKeyHandler{})

	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	ts := NewAPIKeyTokenSource("test", testServer.URL)

	token, err := ts.Token()

	if err != nil {
		t.Fatal(err)
	}

	// Make sure the expiry is correct
	if token.Expiry.Unix() != 1516239022 {
		t.Errorf("token expiry incorrect. Expected 1516239022, got %v", token.Expiry.Unix())
	}
}

func GetWorkingTokenExchange() (string, error) {
	errMap := make(map[string]error)

	for _, url := range tokenExchangeURLs {
		var err error
		if err = testURL(url); err == nil {
			return url, nil
		}
		errMap[url] = err
	}

	var errString string

	for url, err := range errMap {
		errString = errString + fmt.Sprintf("  %v: %v\n", url, err.Error())
	}

	return "", fmt.Errorf("no working token exchanges found:\n%v", errString)
}

func testURL(testURL string) error {
	url, err := url.Parse(testURL)

	if err != nil {
		return fmt.Errorf("could not parse NATS URL: %v. Error: %w", testURL, err)
	}

	dialer := &net.Dialer{
		Timeout: time.Second,
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", net.JoinHostPort(url.Hostname(), url.Port()))

	if err == nil {
		conn.Close()
		return nil
	}

	return err
}
