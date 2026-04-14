package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMCPOAuthMetadataHandler(t *testing.T) {
	scopes := []string{"openid", "profile", "email", "offline_access", "admin:read"}
	handler := NewMCPOAuthMetadataHandler(
		"auth.example.com",
		"https://api.example.com/area51/oauth",
		"https://api.example.com/area51/oauth/register",
		scopes,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/.well-known/oauth-authorization-server/area51/oauth", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}

	if body["issuer"] != "https://api.example.com/area51/oauth" {
		t.Errorf("unexpected issuer: %v", body["issuer"])
	}
	if body["authorization_endpoint"] != "https://auth.example.com/authorize" {
		t.Errorf("unexpected authorization_endpoint: %v", body["authorization_endpoint"])
	}
	if body["token_endpoint"] != "https://auth.example.com/oauth/token" {
		t.Errorf("unexpected token_endpoint: %v", body["token_endpoint"])
	}
	if body["registration_endpoint"] != "https://api.example.com/area51/oauth/register" {
		t.Errorf("unexpected registration_endpoint: %v", body["registration_endpoint"])
	}
	if body["jwks_uri"] != "https://auth.example.com/.well-known/jwks.json" {
		t.Errorf("unexpected jwks_uri: %v", body["jwks_uri"])
	}

	scopesAny, ok := body["scopes_supported"].([]any)
	if !ok {
		t.Fatalf("scopes_supported is not an array: %T", body["scopes_supported"])
	}
	if len(scopesAny) != len(scopes) {
		t.Errorf("expected %d scopes, got %d", len(scopes), len(scopesAny))
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

func TestNewMCPPRMHandler(t *testing.T) {
	handler := NewMCPPRMHandler(
		"https://api.example.com/area51/oauth",
		"https://api.example.com/area51/mcp",
		[]string{"admin:read"},
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/.well-known/oauth-protected-resource/area51/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var body struct {
		Resource               string   `json:"resource"`
		AuthorizationServers   []string `json:"authorization_servers"`
		ScopesSupported        []string `json:"scopes_supported"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
		ClientID               string   `json:"client_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode PRM response: %v", err)
	}

	if body.Resource != "https://api.example.com/area51/mcp" {
		t.Errorf("unexpected resource: %q", body.Resource)
	}
	if len(body.AuthorizationServers) != 1 || body.AuthorizationServers[0] != "https://api.example.com/area51/oauth" {
		t.Errorf("unexpected authorization_servers: %v", body.AuthorizationServers)
	}
	if len(body.ScopesSupported) != 1 || body.ScopesSupported[0] != "admin:read" {
		t.Errorf("unexpected scopes_supported: %v", body.ScopesSupported)
	}
	if len(body.BearerMethodsSupported) != 1 || body.BearerMethodsSupported[0] != "header" {
		t.Errorf("unexpected bearer_methods_supported: %v", body.BearerMethodsSupported)
	}
	if body.ClientID != "" {
		t.Errorf("expected no client_id in PRM, got %q", body.ClientID)
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
