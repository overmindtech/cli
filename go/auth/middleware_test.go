package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/auth0/go-jwt-middleware/v3/validator"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/overmindtech/cli/go/audit"
	log "github.com/sirupsen/logrus"
)

func TestHasScopes(t *testing.T) {
	t.Run("with auth bypassed", func(t *testing.T) {
		t.Parallel()

		ctx := OverrideAuth(context.Background(), WithBypassScopeCheck())

		pass := HasAllScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since auth is bypassed")
		}
	})

	t.Run("with good scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAllScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since `test` scope is present")
		}
	})

	t.Run("with multiple good scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAllScopes(ctx, "test", "foo")

		if !pass {
			t.Error("expected to allow since `test` scope is present")
		}
	})

	t.Run("with bad scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAllScopes(ctx, "baz")

		if pass {
			t.Error("expected to deny since `baz` scope is not present")
		}
	})

	t.Run("with one scope missing", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAllScopes(ctx, "test", "baz")

		if pass {
			t.Error("expected to deny since `baz` scope is not present")
		}
	})

	t.Run("with any scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAnyScopes(ctx, "fail", "foo")

		if !pass {
			t.Error("expected to allow since `foo` scope is present")
		}
	})

	t.Run("without any scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideAuth(context.Background(), WithScope(scope), WithAccount(account))

		pass := HasAnyScopes(ctx, "fail", "fail harder")

		if pass {
			t.Error("expected to deny since no matching scope is present")
		}
	})
}

func TestNewAuthMiddleware(t *testing.T) {
	server, err := NewTestJWTServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	jwksURL := server.Start(ctx)

	defaultConfig := MiddlewareConfig{
		IssuerURL:     jwksURL,
		Auth0Audience: "https://api.overmind.tech",
	}

	bypassHealthConfig := MiddlewareConfig{
		IssuerURL:          jwksURL,
		Auth0Audience:      "https://api.overmind.tech",
		BypassAuthForPaths: regexp.MustCompile("/health"),
	}

	correctAccount := "test"
	correctScope := "test:pass"

	tests := []struct {
		Name         string
		TokenOptions *TestTokenOptions
		ExpectedCode int
		AuthConfig   MiddlewareConfig
		Path         string
	}{
		{
			Name: "with expired token",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(-time.Hour),
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusUnauthorized,
		},
		{
			Name: "with wrong audience",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://something.not.expected"},
				Expiry:   time.Now().Add(time.Hour),
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusUnauthorized,
		},
		{
			Name: "with insufficient scopes",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:fail",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusUnauthorized,
		},
		{
			Name: "with correct scopes but wrong account",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "fail",
					Scope:       "test:pass",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusUnauthorized,
		},
		{
			Name: "with correct scopes and account",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusOK,
		},
		{
			Name: "with the correct scope and many others",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass test:fail foo:bar something",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusOK,
		},
		{
			Name: "with many audiences and many scopes",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech", "https://api.overmind.tech/other"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass test:other",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusOK,
		},
		{
			Name: "with many audiences and one scope",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech", "https://api.overmind.tech/other"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass",
				},
			},
			AuthConfig:   defaultConfig,
			ExpectedCode: http.StatusOK,
		},
		{
			Name: "with good token and some bypassed paths",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass",
				},
			},
			AuthConfig: MiddlewareConfig{
				IssuerURL:          jwksURL,
				Auth0Audience:      "https://api.overmind.tech",
				BypassAuthForPaths: regexp.MustCompile("/health"),
			},
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "with no token on a non-bypassed path",
			Path:         "/",
			AuthConfig:   bypassHealthConfig,
			ExpectedCode: http.StatusUnauthorized,
		},
		{
			Name:         "with no token on a bypassed path",
			Path:         "/health",
			AuthConfig:   bypassHealthConfig,
			ExpectedCode: http.StatusOK,
		},
		{
			Name: "with bad token on a non-bypassed path",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:fail",
				},
			},
			ExpectedCode: http.StatusUnauthorized,
			AuthConfig:   bypassHealthConfig,
		},
		{
			Name: "with bad token on a bypassed path",
			Path: "/health",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:fail",
				},
			},
			ExpectedCode: http.StatusOK,
			AuthConfig:   bypassHealthConfig,
		},
		{
			Name: "with a good token and bypassed auth",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass",
				},
			},
			ExpectedCode: http.StatusOK,
			AuthConfig: MiddlewareConfig{
				IssuerURL:     jwksURL,
				Auth0Audience: "https://api.overmind.tech",
				BypassAuth:    true,
			},
		},
		{
			Name: "with a bad token and bypassed auth",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(-time.Hour), // expired
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:pass",
				},
			},
			ExpectedCode: http.StatusOK,
			AuthConfig: MiddlewareConfig{
				IssuerURL:     jwksURL,
				Auth0Audience: "https://api.overmind.tech",
				BypassAuth:    true,
			},
		},
		{
			Name: "with account override",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "bad",
					Scope:       "test:pass",
				},
			},
			ExpectedCode: http.StatusOK,
			AuthConfig: MiddlewareConfig{
				IssuerURL:       jwksURL,
				Auth0Audience:   "https://api.overmind.tech",
				AccountOverride: &correctAccount,
			},
		},
		{
			Name: "with scope override",
			Path: "/",
			TokenOptions: &TestTokenOptions{
				Audience: []string{"https://api.overmind.tech"},
				Expiry:   time.Now().Add(time.Hour),
				CustomClaims: CustomClaims{
					AccountName: "test",
					Scope:       "test:fail",
				},
			},
			ExpectedCode: http.StatusOK,
			AuthConfig: MiddlewareConfig{
				IssuerURL:     jwksURL,
				Auth0Audience: "https://api.overmind.tech",
				ScopeOverride: &correctScope,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			handler := NewAuthMiddleware(test.AuthConfig, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				// This is a test handler that always does the same thing, it checks
				// that the account is set to the correct value and that the user has
				// the test:pass scope
				if !HasAnyScopes(ctx, "test:pass") {
					w.WriteHeader(http.StatusUnauthorized)
					_, err := w.Write([]byte("missing required scope"))
					if err != nil {
						t.Error(err)
					}
					return
				}

				if ctx.Value(ScopeCheckBypassedContextKey{}) == true {
					// If we are bypassing auth then we don't want to check the account
				} else {
					claims, ok := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
					if !ok {
						w.WriteHeader(http.StatusUnauthorized)
						_, err := fmt.Fprintf(w, "expected *CustomClaims in context, got %T", ctx.Value(CustomClaimsContextKey{}))
						if err != nil {
							t.Error(err)
						}
						return
					}

					if claims.AccountName != "test" {
						w.WriteHeader(http.StatusUnauthorized)
						_, err := fmt.Fprintf(w, "expected account to be 'test', but was '%s'", claims.AccountName)
						if err != nil {
							t.Error(err)
						}
						return
					}
				}
			}))

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, test.Path, nil)
			if err != nil {
				t.Fatal(err)
			}

			if test.TokenOptions != nil {
				// Create a test Token
				token, err := server.GenerateJWT(test.TokenOptions)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			}

			handler.ServeHTTP(rr, req)

			if rr.Code != test.ExpectedCode {
				t.Errorf("expected status code %d, but got %d", test.ExpectedCode, rr.Code)
				t.Error(rr.Body.String())
			}
		})
	}
}

// TestBypassAuthInjectsSubject verifies the BypassAuth code path (local/dev
// environments only — never runs in production where real JWTs provide the
// subject). It ensures a synthetic "auth-bypass" subject is injected into
// CurrentSubjectContextKey so handlers like Area51 job scheduling and feature
// flags work without a JWT.
func TestBypassAuthInjectsSubject(t *testing.T) {
	t.Parallel()

	bypassConfig := MiddlewareConfig{
		BypassAuth: true,
	}

	var capturedSubject string
	handler := NewAuthMiddleware(bypassConfig, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subj, ok := r.Context().Value(CurrentSubjectContextKey{}).(string); ok {
			capturedSubject = subj
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("injects default subject", func(t *testing.T) {
		capturedSubject = ""
		rr := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if capturedSubject != "auth-bypass" {
			t.Errorf("expected subject %q, got %q", "auth-bypass", capturedSubject)
		}
	})

	t.Run("scope check is bypassed", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		scopeHandler := NewAuthMiddleware(bypassConfig, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasAllScopes(r.Context(), "any:scope") {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		scopeHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 (scope check bypassed), got %d", rr.Code)
		}
	})
}

func TestWithSubject(t *testing.T) {
	t.Parallel()

	t.Run("sets subject in context", func(t *testing.T) {
		ctx := OverrideAuth(context.Background(), WithSubject("auth0|user-123"))

		subject, ok := ctx.Value(CurrentSubjectContextKey{}).(string)
		if !ok {
			t.Fatal("expected CurrentSubjectContextKey to be set")
		}
		if subject != "auth0|user-123" {
			t.Errorf("expected subject %q, got %q", "auth0|user-123", subject)
		}
	})

	t.Run("last WithSubject wins", func(t *testing.T) {
		ctx := OverrideAuth(context.Background(),
			WithSubject("first"),
			WithSubject("second"),
		)

		subject, ok := ctx.Value(CurrentSubjectContextKey{}).(string)
		if !ok {
			t.Fatal("expected CurrentSubjectContextKey to be set")
		}
		if subject != "second" {
			t.Errorf("expected subject %q, got %q", "second", subject)
		}
	})

	t.Run("composes with other options", func(t *testing.T) {
		ctx := OverrideAuth(context.Background(),
			WithScope("api:read"),
			WithAccount("test-account"),
			WithSubject("auth0|user-456"),
		)

		subject, ok := ctx.Value(CurrentSubjectContextKey{}).(string)
		if !ok {
			t.Fatal("expected CurrentSubjectContextKey to be set")
		}
		if subject != "auth0|user-456" {
			t.Errorf("expected subject %q, got %q", "auth0|user-456", subject)
		}

		accountName, err := ExtractAccount(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if accountName != "test-account" {
			t.Errorf("expected account %q, got %q", "test-account", accountName)
		}

		if !HasAllScopes(ctx, "api:read") {
			t.Error("expected api:read scope to be present")
		}
	})
}

func TestOverrideAuth(t *testing.T) {
	tests := []struct {
		Name           string
		Options        []OverrideAuthOptionFunc
		HasAllScopes   []string
		HasAccountName string
	}{
		{
			Name: "with account override",
			Options: []OverrideAuthOptionFunc{
				WithAccount("test"),
			},
			HasAccountName: "test",
		},
		{
			Name: "with scope override",
			Options: []OverrideAuthOptionFunc{
				WithScope("test:pass"),
			},
			HasAllScopes: []string{"test:pass"},
		},
		{
			Name: "with account and scope override",
			Options: []OverrideAuthOptionFunc{
				WithAccount("test"),
				WithScope("test:pass"),
			},
			HasAccountName: "test",
			HasAllScopes:   []string{"test:pass"},
		},
		{
			Name: "with account and scope override in reverse order",
			Options: []OverrideAuthOptionFunc{
				WithScope("test:pass"),
				WithAccount("test"),
			},
			HasAccountName: "test",
			HasAllScopes:   []string{"test:pass"},
		},
		{
			Name: "with validated custom claims",
			Options: []OverrideAuthOptionFunc{
				WithValidatedClaims(&validator.ValidatedClaims{
					CustomClaims: &CustomClaims{
						Scope:       "test:pass",
						AccountName: "test",
					},
					RegisteredClaims: validator.RegisteredClaims{
						Issuer:   "https://api.overmind.tech",
						Subject:  "test",
						Audience: []string{"https://api.overmind.tech"},
						ID:       "test",
					},
				}),
			},
			HasAccountName: "test",
			HasAllScopes:   []string{"test:pass"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ctx := context.Background()

			ctx = OverrideAuth(ctx, test.Options...)

			if test.HasAccountName != "" {
				accountName, err := ExtractAccount(ctx)
				if err != nil {
					t.Error(err)
				}

				if accountName != test.HasAccountName {
					t.Errorf("expected account name to be %s, but got %s", test.HasAccountName, accountName)
				}
			}

			for _, scope := range test.HasAllScopes {
				if !HasAllScopes(ctx, scope) {
					t.Errorf("expected to have scope %s, but did not", scope)
				}
			}
		})
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	config := MiddlewareConfig{
		Auth0Domain:   "auth.overmind-demo.com",
		Auth0Audience: "https://api.overmind.tech",
	}

	okHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	handler := NewAuthMiddleware(config, http.HandlerFunc(okHandler))

	// Reduce logging
	log.SetLevel(log.FatalLevel)

	for range b.N {
		// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
		// pass 'nil' as the third parameter.
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		if err != nil {
			b.Fatal(err)
		}

		// Set to a known bad JWT (this JWT is garbage don't freak out)
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InBpQWx1Q1FkQTB4MTNweG1JQzM4dyJ9.eyJodHRwczovL2FwaS5vdmVybWluZC50ZWNoL2FjY291bnQtbmFtZSI6IlRFU1QiLCJpc3MiOiJodHRwczovL29tLWRvZ2Zvb2QuZXUuYXV0aDAuY29tLyIsInN1YiI6ImF1dGgwfFRFU1QiLCJhdWQiOlsiaHR0cHM6Ly9hcGkuZGYub3Zlcm1pbmQtZGVtby5jb20iLCJodHRwczovL29tLWRvZ2Zvb2QuZXUuYXV0aDAuY29tL3VzZXJpbmZvIl0sImlhdCI6MTcxNDA0MjA5MiwiZXhwIjoxNzE0MTI4NDkyLCJzY29wZSI6Im1hbnkgc2NvcGVzIiwiYXpwIjoiVEVTVCJ9.cEEh8jVnEItZel4SoyPybLUg7sArwduCrmSJHMz3YNRfzpRl9lxry39psuDUHFKdgOoNVxUv3Lgm-JWG-9uddCKYOW_zQxEvQvj6o8tcpQkmBZBlc8huG21dLPz7yrPhogVAcApLjdHf1fqii9EHxQegxch9FHlyfF7Xii5t9Hus62l4vdZ5dVWaIuiOLtcbG_hLxl9yqBf5tzN8eEC-Pa1SoAciRPesqH4AARfKyBFBhN774Fu3NzfNtW3wD_ASvnv7aFwzblS8ff5clqdTr2GuuJKdIPcmjQV2LaGSExHg2riCryf5guAhitAuwhugssW__STQmwp8dJmhifs7DA")

		// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			b.Errorf("expected status code %d, but got %d", http.StatusUnauthorized, rr.Code)
		}
	}
}

// Creates a new server that mints real, signed JWTs for testing. It even
// provides its own JWKS endpoint so they can be externally validated. To start
// the JWKS server you should call .Start()
func NewTestJWTServer() (*TestJWTServer, error) {
	// Generate an RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// Wrap this in a JWK object
	jwk := jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     "test-signing-key",
		Algorithm: string(jose.RS256),
	}

	// Create a signer that will sign all of our tokens
	signingKey := jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       jwk,
	}
	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return nil, err
	}

	// Export the public key to be used for validation
	pubJwk := jwk.Public()

	keySet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{pubJwk},
	}

	return &TestJWTServer{
		signer:       signer,
		privateKey:   jwk,
		publicKey:    pubJwk,
		publicKeySet: keySet,
	}, nil
}

// This server is used to mint JWTs for testing purposes. It is basically the
// same as Auth0 when it comes to creating tokens in that it returns a JWKS
// endpoint that can be used to validate the tokens it creates, and the tokens
// use the same algorithm as Auth0
type TestJWTServer struct {
	signer       jose.Signer
	privateKey   jose.JSONWebKey
	publicKey    jose.JSONWebKey
	publicKeySet jose.JSONWebKeySet
	server       *httptest.Server
}

type TestTokenOptions struct {
	Audience []string
	Expiry   time.Time

	CustomClaims
}

func (s *TestJWTServer) GenerateJWT(options *TestTokenOptions) (string, error) {
	builder := jwt.Signed(s.signer)

	builder = builder.Claims(jwt.Claims{
		Issuer:   s.server.URL,
		Subject:  "test",
		Audience: jwt.Audience(options.Audience),
		Expiry:   jwt.NewNumericDate(options.Expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	})

	builder = builder.Claims(options.CustomClaims)

	return builder.Serialize()
}

// Starts the server in the background, the server will exit when the context is
// cancelled. Returns the URL of the server
func (s *TestJWTServer) Start(ctx context.Context) string {
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			// The endpoint tells the validating party where to find the JWKS,
			// this contains our public keys that can be used to validate tokens
			// issued by our server
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintf(w, `{"issuer": %q, "jwks_uri": "%s/.well-known/jwks.json"}`, s.server.URL, s.server.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case "/.well-known/jwks.json":
			// Write the public key set as JSON
			w.Header().Set("Content-Type", "application/json")

			b, err := json.MarshalIndent(s.publicKeySet, "", "  ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}))

	go func() {
		<-ctx.Done()
		s.server.Close()
	}()

	return s.server.URL
}

func TestWithResourceMetadata(t *testing.T) {
	t.Parallel()

	prmURL := "https://api.example.com/.well-known/oauth-protected-resource/area51/mcp"

	t.Run("adds WWW-Authenticate on 401", func(t *testing.T) {
		t.Parallel()
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"JWT is missing."}`))
		})

		handler := WithResourceMetadata(prmURL, inner)
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/area51/mcp", nil)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}

		wwwAuth := rr.Header().Get("WWW-Authenticate")
		expected := `Bearer resource_metadata="` + prmURL + `"`
		if wwwAuth != expected {
			t.Errorf("expected WWW-Authenticate %q, got %q", expected, wwwAuth)
		}
	})

	t.Run("no WWW-Authenticate on 200", func(t *testing.T) {
		t.Parallel()
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := WithResourceMetadata(prmURL, inner)
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/area51/mcp", nil)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		if wwwAuth := rr.Header().Get("WWW-Authenticate"); wwwAuth != "" {
			t.Errorf("expected no WWW-Authenticate header, got %q", wwwAuth)
		}
	})

	t.Run("no WWW-Authenticate on 403", func(t *testing.T) {
		t.Parallel()
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})

		handler := WithResourceMetadata(prmURL, inner)
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/area51/mcp", nil)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rr.Code)
		}

		if wwwAuth := rr.Header().Get("WWW-Authenticate"); wwwAuth != "" {
			t.Errorf("expected no WWW-Authenticate header on 403, got %q", wwwAuth)
		}
	})

	t.Run("implicit 200 from Write without WriteHeader", func(t *testing.T) {
		t.Parallel()
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		handler := WithResourceMetadata(prmURL, inner)
		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/area51/mcp", nil)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		if wwwAuth := rr.Header().Get("WWW-Authenticate"); wwwAuth != "" {
			t.Errorf("expected no WWW-Authenticate header, got %q", wwwAuth)
		}
	})
}

func TestConnectErrorHandling(t *testing.T) {
	// Create a test JWT server
	server, err := NewTestJWTServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()

	jwksURL := server.Start(ctx)

	// Create the middleware
	handler := NewAuthMiddleware(MiddlewareConfig{
		Auth0Domain:   "",
		Auth0Audience: "test",
		IssuerURL:     jwksURL,
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		Name               string
		ContentType        string
		ExpectJSONResponse bool
		ExpectContentType  string
	}{
		{
			Name:               "Regular JSON request without auth",
			ContentType:        "application/json",
			ExpectJSONResponse: true,
			ExpectContentType:  "application/json",
		},
		{
			Name:               "Connect proto request without auth",
			ContentType:        "application/connect+proto",
			ExpectJSONResponse: false,
			ExpectContentType:  "",
		},
		{
			Name:               "Connect json request without auth",
			ContentType:        "application/connect+json",
			ExpectJSONResponse: false,
			ExpectContentType:  "",
		},
		{
			Name:               "gRPC base request without auth",
			ContentType:        "application/grpc",
			ExpectJSONResponse: false,
			ExpectContentType:  "",
		},
		{
			Name:               "gRPC proto request without auth",
			ContentType:        "application/grpc+proto",
			ExpectJSONResponse: false,
			ExpectContentType:  "",
		},
		{
			Name:               "gRPC json request without auth",
			ContentType:        "application/grpc+json",
			ExpectJSONResponse: false,
			ExpectContentType:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set the Content-Type header
			req.Header.Set("Content-Type", test.ContentType)

			// Don't set any auth token, so it will fail auth
			handler.ServeHTTP(rr, req)

			// Should return 401 Unauthorized
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status code %d, but got %d", http.StatusUnauthorized, rr.Code)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if test.ExpectContentType != contentType {
				t.Errorf("expected Content-Type header to be '%s', but got '%s'", test.ExpectContentType, contentType)
			}

			// Check if response has JSON body
			hasJSONBody := len(rr.Body.Bytes()) > 0 && contentType == "application/json"
			if test.ExpectJSONResponse != hasJSONBody {
				t.Errorf("expected JSON response: %v, but got: %v (body length: %d)", test.ExpectJSONResponse, hasJSONBody, len(rr.Body.Bytes()))
			}
		})
	}
}

func TestAuthMiddleware_PopulatesAuditData(t *testing.T) {
	server, err := NewTestJWTServer()
	if err != nil {
		t.Fatal(err)
	}

	jwksURL := server.Start(t.Context())

	discardLogger := log.New()
	discardLogger.SetOutput(io.Discard)

	t.Run("populates audit data from JWT", func(t *testing.T) {
		var capturedAD *audit.AuditData

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedAD = audit.AuditDataFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := audit.NewAuditMiddleware(discardLogger)(
			NewAuthMiddleware(MiddlewareConfig{
				IssuerURL:     jwksURL,
				Auth0Audience: "https://api.overmind.tech",
			}, inner),
		)

		token, err := server.GenerateJWT(&TestTokenOptions{
			Audience: []string{"https://api.overmind.tech"},
			Expiry:   time.Now().Add(time.Hour),
			CustomClaims: CustomClaims{
				AccountName: "acme-corp",
				Scope:       "read:items write:items",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if capturedAD == nil {
			t.Fatal("expected audit data to be present in context")
		}
		if capturedAD.Subject != "test" {
			t.Errorf("expected subject 'test', got %q", capturedAD.Subject)
		}
		if capturedAD.AccountName != "acme-corp" {
			t.Errorf("expected account 'acme-corp', got %q", capturedAD.AccountName)
		}
		if capturedAD.Scopes != "read:items write:items" {
			t.Errorf("expected scopes 'read:items write:items', got %q", capturedAD.Scopes)
		}
	})

	t.Run("populates audit data with account override", func(t *testing.T) {
		var capturedAD *audit.AuditData

		override := "override-acme"
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedAD = audit.AuditDataFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := audit.NewAuditMiddleware(discardLogger)(
			NewAuthMiddleware(MiddlewareConfig{
				IssuerURL:       jwksURL,
				Auth0Audience:   "https://api.overmind.tech",
				AccountOverride: &override,
			}, inner),
		)

		token, err := server.GenerateJWT(&TestTokenOptions{
			Audience: []string{"https://api.overmind.tech"},
			Expiry:   time.Now().Add(time.Hour),
			CustomClaims: CustomClaims{
				AccountName: "original-acme",
				Scope:       "read:items",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if capturedAD == nil {
			t.Fatal("expected audit data to be present in context")
		}
		if capturedAD.AccountName != "override-acme" {
			t.Errorf("expected overridden account 'override-acme', got %q", capturedAD.AccountName)
		}
	})

	t.Run("works without audit context", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := NewAuthMiddleware(MiddlewareConfig{
			IssuerURL:     jwksURL,
			Auth0Audience: "https://api.overmind.tech",
		}, inner)

		token, err := server.GenerateJWT(&TestTokenOptions{
			Audience: []string{"https://api.overmind.tech"},
			Expiry:   time.Now().Add(time.Hour),
			CustomClaims: CustomClaims{
				AccountName: "acme-corp",
				Scope:       "read:items",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 (no panic without audit context), got %d", rr.Code)
		}
	})
}
