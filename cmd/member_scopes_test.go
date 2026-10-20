package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/overmindtech/cli/go/auth"
	"github.com/overmindtech/cli/go/cliauth"
	"golang.org/x/oauth2"
)

func TestMemberWorkflowScopes(t *testing.T) {
	const memberScopes = "account:read changes:read changes:write config:read explore:read request:receive request:send reverselink:request source:read sources:read"
	for name, required := range map[string][]string{"explore": localDiscoveryScopes(), "terraform": terraformWorkflowScopes()} {
		t.Run(name, func(t *testing.T) {
			for _, scopes := range append([]string{memberScopes}, required...) {
				granted := memberScopes
				want := true
				if scopes != memberScopes {
					granted = strings.ReplaceAll(memberScopes, scopes, "")
					want = false
				}
				payload, err := json.Marshal(auth.CustomClaims{Scope: granted})
				if err != nil {
					t.Fatal(err)
				}
				token := &oauth2.Token{AccessToken: fmt.Sprintf(".%s.", base64.RawURLEncoding.EncodeToString(payload))}
				got, missing, err := cliauth.HasScopesFlexible(token, required)
				if err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Errorf("granted %q: expected allowed=%v, got %v (missing %v)", granted, want, got, missing)
				}
			}
		})
	}
}
