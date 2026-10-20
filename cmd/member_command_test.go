//go:build !race

package cmd

// Command smoke tests exercise pterm's animated UI, which has pre-existing data
// races in SpinnerPrinter. Run these without -race; the workflow scope tests and
// backend authorization tests also run separately with race detection.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Exercise command authentication against a local API rather than only checking
// the shared scope lists. No Auth0, cloud credentials or NATS server is involved.
func TestMemberCommandAuthentication(t *testing.T) {
	const memberScopes = "account:read changes:read changes:write config:read explore:read request:receive request:send reverselink:request source:read sources:read"
	for _, command := range []string{"terraform", "explore"} {
		for _, allowed := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/allowed=%v", command, allowed), func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				scopes := memberScopes
				if !allowed {
					scopes = "account:read"
				}
				claims, err := json.Marshal(map[string]any{"scope": scopes, "sub": "member", "exp": time.Now().Add(time.Hour).Unix(), "https://api.overmind.tech/account-name": "test"})
				if err != nil {
					t.Fatal(err)
				}
				// ensureToken parses this JWT but signature verification belongs to the server.
				token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." + base64.RawURLEncoding.EncodeToString(claims) + ".AA"
				mux := http.NewServeMux()
				var exchanged atomic.Bool
				mux.Handle(sdpconnect.ApiKeyServiceExchangeKeyForTokenProcedure, connect.NewUnaryHandler(sdpconnect.ApiKeyServiceExchangeKeyForTokenProcedure, func(_ context.Context, _ *connect.Request[sdp.ExchangeKeyForTokenRequest]) (*connect.Response[sdp.ExchangeKeyForTokenResponse], error) {
					exchanged.Store(true)
					return connect.NewResponse(&sdp.ExchangeKeyForTokenResponse{AccessToken: token}), nil
				}))
				server := httptest.NewServer(mux)
				defer server.Close()
				mux.HandleFunc("/api/public/instance-data", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(map[string]string{"api_url": server.URL, "nats_url": "nats://127.0.0.1:4222"}); err != nil {
						t.Error(err)
					}
				})
				viper.Set("app", server.URL)
				viper.Set("api-key", "ovm_api_test")
				viper.Set("timeout", "5s")
				viper.Set("only-use-managed-sources", true)
				cmd := &cobra.Command{Use: command}
				cmd.SetContext(t.Context())
				if command == "terraform" {
					_, _, got, cleanup, err := StartSources(t.Context(), cmd, nil)
					if cleanup != nil {
						cleanup()
					}
					if allowed && (err != nil || got == nil) {
						t.Fatalf("expected Member authentication success, got token=%v, error=%v", got != nil, err)
					}
					if !allowed && err == nil {
						t.Error("expected authentication to reject missing workflow scopes")
					}
				} else {
					// A missing snapshot stops after authentication, before starting a source
					// or opening the browser. The wrapped filesystem error proves that the
					// Member passed the real Explore command's login requirements.
					t.Setenv("SNAPSHOT_SOURCE", filepath.Join(t.TempDir(), "missing.json"))
					err := Explore(cmd, nil)
					if allowed && !errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("expected snapshot file error after Member login, got %v", err)
					}
					if !allowed && (err == nil || errors.Is(err, fs.ErrNotExist)) {
						t.Errorf("expected login rejection before source startup, got %v", err)
					}
				}
				if !exchanged.Load() {
					t.Error("expected command to exchange credentials with local API")
				}
			})
		}
	}
}
