package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jwksURL := server.Start(ctx)

	defaultConfig := AuthConfig{
		IssuerURL:     jwksURL,
		Auth0Audience: "https://api.overmind.tech",
	}

	bypassHealthConfig := AuthConfig{
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
		AuthConfig   AuthConfig
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
			AuthConfig: AuthConfig{
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
			AuthConfig: AuthConfig{
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
			AuthConfig: AuthConfig{
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
			AuthConfig: AuthConfig{
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
			AuthConfig: AuthConfig{
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
	config := AuthConfig{
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
			_, err := fmt.Fprintf(w, `{"jwks_uri": "%s/.well-known/jwks.json"}`, s.server.URL)
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
