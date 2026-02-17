// This file is adapted from https://gist.github.com/ahmetb/548059cdbf12fb571e4e2f1e29c48997

package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"k8s.io/client-go/rest"
)

var (
	googleScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email"}
)

const (
	GoogleAuthPlugin = "custom_gcp" // so that this is different than "gcp" that's already in client-go tree.
)

func init() {
	if err := rest.RegisterAuthProviderPlugin(GoogleAuthPlugin, newGoogleAuthProvider); err != nil {
		log.Fatalf("Failed to register %s auth plugin: %v", GoogleAuthPlugin, err)
	}
}

var _ rest.AuthProvider = &googleAuthProvider{}

type googleAuthProvider struct {
	tokenSource oauth2.TokenSource
}

func (g *googleAuthProvider) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &oauth2.Transport{
		Base:   rt,
		Source: g.tokenSource,
	}
}
func (g *googleAuthProvider) Login() error { return nil }

func newGoogleAuthProvider(addr string, config map[string]string, persister rest.AuthProviderConfigPersister) (rest.AuthProvider, error) {
	scopes := googleScopes
	scopesCfg, found := config["scopes"]
	if found {
		scopes = strings.Split(scopesCfg, " ")
	}
	ts, err := google.DefaultTokenSource(context.Background(), scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to create google token source: %w", err)
	}
	return &googleAuthProvider{tokenSource: ts}, nil
}
