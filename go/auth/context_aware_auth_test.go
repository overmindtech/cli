package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextAwareAuthTransport_InjectsToken(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
	}))
	defer ts.Close()

	client := NewContextAwareAuthClient(ts.Client())

	ctx := context.WithValue(context.Background(), UserTokenContextKey{}, "test-jwt-token")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if capturedAuth != "Bearer test-jwt-token" {
		t.Errorf("expected 'Bearer test-jwt-token', got %q", capturedAuth)
	}
}

func TestContextAwareAuthTransport_NoToken(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
	}))
	defer ts.Close()

	client := NewContextAwareAuthClient(ts.Client())

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if capturedAuth != "" {
		t.Errorf("expected empty auth header, got %q", capturedAuth)
	}
}

func TestContextAwareAuthTransport_DifferentTokensPerRequest(t *testing.T) {
	var capturedTokens []string
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTokens = append(capturedTokens, r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	client := NewContextAwareAuthClient(ts.Client())

	for _, token := range []string{"token-a", "token-b"} {
		ctx := context.WithValue(context.Background(), UserTokenContextKey{}, token)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp.Body.Close()
	}

	if len(capturedTokens) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(capturedTokens))
	}
	if capturedTokens[0] != "Bearer token-a" {
		t.Errorf("first request: expected 'Bearer token-a', got %q", capturedTokens[0])
	}
	if capturedTokens[1] != "Bearer token-b" {
		t.Errorf("second request: expected 'Bearer token-b', got %q", capturedTokens[1])
	}
}
