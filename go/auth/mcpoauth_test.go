package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNewMCPOAuthMetadataHandler_RouteFamilyAliasContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		issuer                string
		registrationEndpoint  string
		authorizationEndpoint string
		tokenEndpoint         string
		scopes                []string
		aliases               []string
		forbidden             []string
	}{
		{
			name:                  "canonical Until",
			issuer:                "https://until.example.com/oauth",
			registrationEndpoint:  "https://until.example.com/oauth/register",
			authorizationEndpoint: "https://until.example.com/oauth/authorize",
			tokenEndpoint:         "https://until.example.com/oauth/token",
			scopes:                []string{"account:read", "until:read", "until:write"},
			aliases: []string{
				"/.well-known/oauth-authorization-server/oauth",
				"/oauth/.well-known/oauth-authorization-server",
				"/.well-known/openid-configuration/oauth",
				"/oauth/.well-known/openid-configuration",
			},
			forbidden: []string{"/brent/"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := NewMCPOAuthMetadataHandler(
				"auth.example.com",
				test.issuer,
				test.registrationEndpoint,
				test.scopes,
				test.authorizationEndpoint,
				test.tokenEndpoint,
			)
			mux := http.NewServeMux()
			for _, alias := range test.aliases {
				mux.Handle(alias, handler)
			}

			wantBody := map[string]any{
				"issuer":                 test.issuer,
				"authorization_endpoint": test.authorizationEndpoint,
				"token_endpoint":         test.tokenEndpoint,
				"registration_endpoint":  test.registrationEndpoint,
				"jwks_uri":               "https://auth.example.com/.well-known/jwks.json",
				"userinfo_endpoint":      "https://auth.example.com/userinfo",
				"revocation_endpoint":    "https://auth.example.com/oauth/revoke",
				"response_types_supported": []any{
					"code",
				},
				"grant_types_supported": []any{
					"authorization_code",
					"refresh_token",
				},
				"code_challenge_methods_supported": []any{
					"S256",
				},
				"token_endpoint_auth_methods_supported": []any{
					"none",
				},
				"scopes_supported": stringsToAny(test.scopes),
			}

			var firstAliasBody []byte
			for _, alias := range test.aliases {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, alias, nil))
				if got, want := rec.Code, http.StatusOK; got != want {
					t.Fatalf("%s status = %d, want %d", alias, got, want)
				}
				if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
					t.Errorf("%s Content-Type = %q, want %q", alias, got, want)
				}

				bodyBytes := rec.Body.Bytes()
				if firstAliasBody == nil {
					firstAliasBody = bytes.Clone(bodyBytes)
				} else if !bytes.Equal(bodyBytes, firstAliasBody) {
					t.Errorf("%s body differs byte-for-byte from first alias: got %q, want %q", alias, bodyBytes, firstAliasBody)
				}

				var body map[string]any
				if err := json.Unmarshal(bodyBytes, &body); err != nil {
					t.Fatalf("%s decode metadata: %v", alias, err)
				}
				if !reflect.DeepEqual(body, wantBody) {
					t.Errorf("%s body = %#v, want %#v", alias, body, wantBody)
				}
				for _, forbidden := range test.forbidden {
					if strings.Contains(string(bodyBytes), forbidden) {
						t.Errorf("%s canonical body contains legacy value %q: %s", alias, forbidden, bodyBytes)
					}
				}
			}
		})
	}
}

func TestNewMCPOAuthMetadataHandler_DefaultAuth0Endpoints(t *testing.T) {
	t.Parallel()

	handler := NewMCPOAuthMetadataHandler(
		"auth.example.com",
		"https://api.example.com/area51/oauth",
		"https://api.example.com/area51/oauth/register",
		[]string{"openid"},
		"",
		"",
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metadata", nil))

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if got, want := body["authorization_endpoint"], "https://auth.example.com/authorize"; got != want {
		t.Errorf("authorization_endpoint = %v, want %q", got, want)
	}
	if got, want := body["token_endpoint"], "https://auth.example.com/oauth/token"; got != want {
		t.Errorf("token_endpoint = %v, want %q", got, want)
	}
}

func TestNewMCPDCRHandler(t *testing.T) {
	handler := NewMCPDCRHandler("test-client-id")

	reqBody := `{"redirect_uris":["http://127.0.0.1/callback"],"client_name":"Test Client"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/area51/oauth/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var body struct {
		ClientID                string   `json:"client_id"`
		RedirectURIs            []string `json:"redirect_uris"`
		ClientName              string   `json:"client_name"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode DCR response: %v", err)
	}

	if body.ClientID != "test-client-id" {
		t.Errorf("unexpected client_id: %q", body.ClientID)
	}
	if body.TokenEndpointAuthMethod != "none" {
		t.Errorf("unexpected token_endpoint_auth_method: %q", body.TokenEndpointAuthMethod)
	}
	if len(body.RedirectURIs) != 1 || body.RedirectURIs[0] != "http://127.0.0.1/callback" {
		t.Errorf("unexpected redirect_uris: %v", body.RedirectURIs)
	}
	if body.ClientName != "Test Client" {
		t.Errorf("unexpected client_name: %q", body.ClientName)
	}
}

func TestNewMCPDCRHandler_MethodNotAllowed(t *testing.T) {
	handler := NewMCPDCRHandler("test-client-id")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/area51/oauth/register", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestNewMCPDCRHandler_FiltersNonLocalhostRedirects(t *testing.T) {
	handler := NewMCPDCRHandler("test-client-id")

	reqBody := `{"redirect_uris":["http://127.0.0.1/callback","https://evil.com/callback","http://localhost:3000/callback"]}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var body struct {
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(body.RedirectURIs) != 2 {
		t.Fatalf("expected 2 safe redirect URIs, got %d: %v", len(body.RedirectURIs), body.RedirectURIs)
	}
}

func TestNewMCPPRMHandler_RouteFamilyContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		path                string
		authorizationServer string
		resource            string
		scopes              []string
		forbidden           []string
	}{
		{
			name:                "canonical Until",
			path:                "/.well-known/oauth-protected-resource/mcp",
			authorizationServer: "https://until.example.com/oauth",
			resource:            "https://until.example.com/mcp",
			scopes:              []string{"account:read", "until:read", "until:write"},
			forbidden: []string{"/brent/"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := NewMCPPRMHandler(test.authorizationServer, test.resource, test.scopes)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil))

			if got, want := rec.Code, http.StatusOK; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}
			if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}

			bodyBytes := rec.Body.Bytes()
			var body map[string]any
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				t.Fatalf("decode PRM response: %v", err)
			}
			wantBody := map[string]any{
				"resource":                 test.resource,
				"authorization_servers":    []any{test.authorizationServer},
				"scopes_supported":         stringsToAny(test.scopes),
				"bearer_methods_supported": []any{"header"},
			}
			if !reflect.DeepEqual(body, wantBody) {
				t.Errorf("body = %#v, want %#v", body, wantBody)
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(string(bodyBytes), forbidden) {
					t.Errorf("canonical body contains legacy value %q: %s", forbidden, bodyBytes)
				}
			}
		})
	}
}

func TestNewMCPOAuthMetadataHandler_Overrides(t *testing.T) {
	scopes := []string{"openid"}
	handler := NewMCPOAuthMetadataHandler(
		"auth.example.com",
		"https://api.example.com/oauth",
		"https://api.example.com/oauth/register",
		scopes,
		"https://api.example.com/oauth/authorize",
		"https://api.example.com/oauth/token",
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metadata", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["authorization_endpoint"] != "https://api.example.com/oauth/authorize" {
		t.Errorf("authorization_endpoint = %v", body["authorization_endpoint"])
	}
	if body["token_endpoint"] != "https://api.example.com/oauth/token" {
		t.Errorf("token_endpoint = %v", body["token_endpoint"])
	}
}

func TestIsAllowedMCPRedirect(t *testing.T) {
	tests := []struct {
		uri  string
		want bool
	}{
		{"cursor://anysphere.cursor-mcp/oauth/callback", true},
		{"https://www.cursor.com/agents/mcp/oauth/callback", true},
		{"http://127.0.0.1/callback", true},
		{"https://evil.com/callback", false},
	}
	for _, tt := range tests {
		if got := IsAllowedMCPRedirect(tt.uri); got != tt.want {
			t.Errorf("IsAllowedMCPRedirect(%q) = %v, want %v", tt.uri, got, tt.want)
		}
	}
}

func TestIsLocalhostRedirect(t *testing.T) {
	tests := []struct {
		uri  string
		want bool
	}{
		{"http://127.0.0.1/callback", true},
		{"http://localhost:3000/callback", true},
		{"http://[::1]:8080/callback", true},
		{"https://evil.com/callback", false},
		{"https://example.com", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		got := IsLocalhostRedirect(tt.uri)
		if got != tt.want {
			t.Errorf("IsLocalhostRedirect(%q) = %v, want %v", tt.uri, got, tt.want)
		}
	}
}

func stringsToAny(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
