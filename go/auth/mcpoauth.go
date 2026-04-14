package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// NewMCPOAuthMetadataHandler returns an HTTP handler that serves OAuth 2.0
// Authorization Server Metadata (RFC 8414) for an MCP endpoint.
//
// Instead of proxying Auth0's metadata at runtime, it constructs a static
// document that points authorization_endpoint and token_endpoint to Auth0
// while advertising our own registration_endpoint for Dynamic Client
// Registration (RFC 7591). This lets MCP clients like Cursor discover
// the client_id automatically without any user configuration.
//
// scopes should include both the standard OIDC scopes and any
// application-specific scopes (e.g. "admin:read", "changes:read").
func NewMCPOAuthMetadataHandler(auth0Domain, issuerURL, registrationEndpointURL string, scopes []string) http.Handler {
	metadata := map[string]any{
		"issuer":                 issuerURL,
		"authorization_endpoint": fmt.Sprintf("https://%s/authorize", auth0Domain),
		"token_endpoint":         fmt.Sprintf("https://%s/oauth/token", auth0Domain),
		"registration_endpoint":  registrationEndpointURL,

		"jwks_uri":            fmt.Sprintf("https://%s/.well-known/jwks.json", auth0Domain),
		"userinfo_endpoint":   fmt.Sprintf("https://%s/userinfo", auth0Domain),
		"revocation_endpoint": fmt.Sprintf("https://%s/oauth/revoke", auth0Domain),

		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      scopes,
	}

	body, _ := json.Marshal(metadata) //nolint:errchkjson // static map of strings/slices, cannot fail

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
}

// NewMCPDCRHandler returns an HTTP handler that implements a minimal OAuth 2.0
// Dynamic Client Registration (RFC 7591) endpoint. It always returns the
// same pre-configured Auth0 client_id since all MCP clients share a single
// public OAuth application.
//
// Per RFC 7591 Section 3.2, the response echoes back the registered client
// metadata including redirect_uris from the request.
func NewMCPDCRHandler(clientID string) http.Handler {
	type dcrRequest struct {
		RedirectURIs []string `json:"redirect_uris"`
		ClientName   string   `json:"client_name"`
	}

	type dcrResponse struct {
		ClientID                string   `json:"client_id"`
		RedirectURIs            []string `json:"redirect_uris"`
		ClientName              string   `json:"client_name,omitempty"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limited := http.MaxBytesReader(w, r.Body, 64<<10)
		var req dcrRequest
		if err := json.NewDecoder(limited).Decode(&req); err != nil {
			req = dcrRequest{}
		}
		_ = limited.Close()

		// Don't echo back arbitrary redirect_uris; Auth0 enforces the
		// registered set during token exchange, but echoing unchecked URIs
		// could mislead clients that trust the DCR response blindly.
		// Instead, return only localhost URIs which are the standard
		// callback for native/public OAuth clients per RFC 8252.
		safeURIs := make([]string, 0, len(req.RedirectURIs))
		for _, uri := range req.RedirectURIs {
			if IsLocalhostRedirect(uri) {
				safeURIs = append(safeURIs, uri)
			}
		}

		resp := dcrResponse{
			ClientID:                clientID,
			RedirectURIs:            safeURIs,
			ClientName:              req.ClientName,
			TokenEndpointAuthMethod: "none",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// NewMCPPRMHandler returns an http.Handler that serves the OAuth 2.0 Protected
// Resource Metadata (RFC 9728) JSON document for an MCP endpoint. No
// authentication is required.
//
// authorizationServerURL is the issuer URL of the OAuth metadata endpoint (not
// the raw Auth0 domain). MCP clients use this to discover the authorization
// and token endpoints, as well as the Dynamic Client Registration endpoint.
func NewMCPPRMHandler(authorizationServerURL, resourceURL string, scopes []string) http.Handler {
	type prmResponse struct {
		Resource               string   `json:"resource"`
		AuthorizationServers   []string `json:"authorization_servers"`
		ScopesSupported        []string `json:"scopes_supported"`
		BearerMethodsSupported []string `json:"bearer_methods_supported"`
	}

	resp := prmResponse{
		Resource:               resourceURL,
		AuthorizationServers:   []string{authorizationServerURL},
		ScopesSupported:        scopes,
		BearerMethodsSupported: []string{"header"},
	}

	body, _ := json.Marshal(resp) //nolint:errchkjson // static struct of strings, cannot fail

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// IsLocalhostRedirect returns true if the URI is a loopback redirect, which is
// the standard callback for native/public OAuth clients (RFC 8252 Section 7.3).
func IsLocalhostRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}
