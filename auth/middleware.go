package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ScopeCheckBypassedContextKey is a key that is stored in the request context
// when scope checking is actively being bypassed, e.g. in development. When
// this is set the `HasScopes()` function will always return true, and can be
// set using the `WithBypassScopeCheck()` middleware.
type ScopeCheckBypassedContextKey struct{}

// CustomClaimsContextKey is the key that is used to store the custom claims
// from the JWT
type CustomClaimsContextKey struct{}

// AccountNameContextKey is the key that is used to store the currently acting
// account name
type AccountNameContextKey struct{}

// UserTokenContextKey is the key that is used to store the full JWT token of the user
type UserTokenContextKey struct{}

// CurrentSubjectContextKey is the key that is used to store the current subject attribute.
// This will be the auth0 `user_id` from the tokens `sub` claim.
type CurrentSubjectContextKey struct{}

// AuthConfig Configuration for the auth middleware
type AuthConfig struct {
	Auth0Domain   string
	Auth0Audience string
	// The names of the cookies that will be used to authenticate, these will be
	// checked in order with the first one that is found being used
	AuthCookieNames []string

	// Use this to specify the full issuer URL for validating the JWTs. This
	// should only be used if we aren't using Auth0 as a source for tokens (such
	// as in testing). Auth0Domain will take precedence if both are set.
	IssuerURL string

	// Bypasses all auth checks, meaning that HasScopes() will always return
	// true. This should be used in conjunction with the `AccountOverride` field
	// since there won't be a token to parse the account from
	BypassAuth bool

	// Bypasses auth for the given paths. This is a regular expression that is
	// matched against the path of the request. If the regex matches then the
	// request will be allowed through without auth. This should be used with
	// `AccountOverride` in order to avoid the required context values not being
	// set and therefore causing issues (probably nil pointer panics)
	BypassAuthForPaths *regexp.Regexp

	// Overrides the account name stored in the CustomClaimsContextKey
	AccountOverride *string

	// Overrides the scope stored in the CustomClaimsContextKey
	ScopeOverride *string
}

// HasScopes compatibility alias for HasAllScopes
func HasScopes(ctx context.Context, requiredScopes ...string) bool {
	return HasAllScopes(ctx, requiredScopes...)
}

// HasAllScopes checks that the authenticated user in the request context has all the
// required scopes. If auth has been bypassed, this will always return true
func HasAllScopes(ctx context.Context, requiredScopes ...string) bool {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.StringSlice("ovm.auth.requiredScopes.all", requiredScopes),
	)

	if ctx.Value(ScopeCheckBypassedContextKey{}) == true {
		// this is always set when auth is bypassed
		// set it here again to capture non-standard auth configs
		span.SetAttributes(attribute.Bool("ovm.auth.bypass", true))

		// Bypass all auth
		return true
	}

	claims, ok := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
	if !ok {
		span.SetAttributes(attribute.String("ovm.auth.missingClaims", "all"))
		return false
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			span.SetAttributes(attribute.String("ovm.auth.missingClaims", scope))
			return false
		}
	}
	return true
}

// HasAnyScopes checks that the authenticated user in the request context has any of the
// required scopes. If auth has been bypassed, this will always return true
func HasAnyScopes(ctx context.Context, requiredScopes ...string) bool {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.StringSlice("ovm.auth.requiredScopes.any", requiredScopes),
	)

	if ctx.Value(ScopeCheckBypassedContextKey{}) == true {
		// this is always set when auth is bypassed
		// set it here again to capture non-standard auth configs
		span.SetAttributes(attribute.Bool("ovm.auth.bypass", true))

		// Bypass all auth
		return true
	}

	claims, ok := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
	if !ok {
		span.SetAttributes(attribute.String("ovm.auth.missingClaims", "all"))
		return false
	}

	span.SetAttributes(
		attribute.String("ovm.auth.tokenScopes", claims.Scope),
	)

	for _, scope := range requiredScopes {
		if claims.HasScope(scope) {
			span.SetAttributes(attribute.String("ovm.auth.usedClaim", scope))
			return true
		}
	}
	return false
}

var ErrNoClaims = errors.New("error extracting claims from token")

// ExtractAccount Extracts the account name from a context
func ExtractAccount(ctx context.Context) (string, error) {
	claims := ctx.Value(CustomClaimsContextKey{})

	if claims == nil {
		return "", ErrNoClaims
	}

	return claims.(*CustomClaims).AccountName, nil
}

// NewAuthMiddleware Creates new auth middleware. The options allow you to
// bypass the authentication process or not, but either way this middleware will
// set the `CustomClaimsContextKey` in the request context which allows you to
// use the `HasScopes()` function to check the scopes without having to worry
// about whether the server is using auth or not.
//
// If auth is not bypassed, then tokens will be validated using Auth0 and
// therefore the following environment variables must be set: AUTH0_DOMAIN,
// AUTH0_AUDIENCE. If cookie auth is intended to be used, then AUTH_COOKIE_NAME
// must also be set.
func NewAuthMiddleware(config AuthConfig, next http.Handler) http.Handler {
	processOverrides := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options := []OverrideAuthOptionFunc{}

		if config.ScopeOverride != nil {
			options = append(options, WithScope(*config.ScopeOverride))
		}

		if config.AccountOverride != nil {
			options = append(options, WithAccount(*config.AccountOverride))
		}

		ctx := r.Context()
		if len(options) > 0 {
			ctx = OverrideAuth(r.Context(), options...)
		}

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	})

	return ensureValidTokenHandler(config, processOverrides)
}

type OverrideAuthOptionFunc func(ctx context.Context) context.Context

// Sets the scope in the context to the given value. This should be the value
// that would be embedded directly in the token, with each scope being separated
// by a space.
func WithScope(scope string) OverrideAuthOptionFunc {
	return withCustomClaims(func(claims *CustomClaims) {
		claims.Scope = scope
	})
}

// Sets the account in the context to the given value.
func WithAccount(account string) OverrideAuthOptionFunc {
	return withCustomClaims(func(claims *CustomClaims) {
		claims.AccountName = account
	})
}

// Sets the auth info in the context directly from the validated claims produced
// by the `github.com/auth0/go-jwt-middleware/v2/validator` package. This is
// essentially what the middleware already does when receiving a request, and
// therefore should only be used in exceptional circumstances, like testing, when the
// middleware is not being used.
//
// If this is being used, there is no need to use the `WithScope` or `WithAccount`
// options as the claims will be extracted directly from the validated claims.
func WithValidatedClaims(claims *validator.ValidatedClaims) OverrideAuthOptionFunc {
	return func(ctx context.Context) context.Context {
		customClaims := claims.CustomClaims.(*CustomClaims)
		ctx = context.WithValue(ctx, jwtmiddleware.ContextKey{}, claims)
		ctx = context.WithValue(ctx, CustomClaimsContextKey{}, customClaims)
		ctx = context.WithValue(ctx, CurrentSubjectContextKey{}, claims.RegisteredClaims.Subject)
		ctx = context.WithValue(ctx, AccountNameContextKey{}, customClaims.AccountName)
		return ctx
	}
}

// Bypasses the scope check, meaning that `HasScopes()` and `HasAllScopes` will
// always return true. This is useful for testing.
func WithBypassScopeCheck() OverrideAuthOptionFunc {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, ScopeCheckBypassedContextKey{}, true)
	}
}

// Overrides the authentication that is currently stored in the context. This
// can only be used within a single process, and doesn't mean that the overrides
// set here will be passed on if you are using `NewAuthenticatedClient` to pass
// through auth. It is however useful for testing, or for calling other handlers
// within the same process.
func OverrideAuth(ctx context.Context, opts ...OverrideAuthOptionFunc) context.Context {
	for _, opt := range opts {
		ctx = opt(ctx)
	}
	return ctx
}

func withCustomClaims(modify func(*CustomClaims)) OverrideAuthOptionFunc {
	return func(ctx context.Context) context.Context {
		i := ctx.Value(CustomClaimsContextKey{})
		var claims *CustomClaims
		var newClaims CustomClaims
		var ok bool

		if claims, ok = i.(*CustomClaims); ok {
			// clone out the values to avoid sharing
			newClaims = *claims
		}

		modify(&newClaims)

		// Store the new claims in the context
		ctx = context.WithValue(ctx, CustomClaimsContextKey{}, &newClaims)
		ctx = context.WithValue(ctx, AccountNameContextKey{}, newClaims.AccountName)

		return ctx
	}
}

// ensureValidTokenHandler is a middleware that will check the validity of our
// JWT.
//
// This will fail if all of Auth0Domain, Auth0Audience and AuthCookieName are
// empty.
//
// This middleware also extract custom claims form the token and stores them in
// CustomClaimsContextKey
func ensureValidTokenHandler(config AuthConfig, next http.Handler) http.Handler {
	if config.Auth0Domain == "" && config.IssuerURL == "" && config.Auth0Audience == "" {
		log.Fatalf("Auth0 configuration is missing")
	}

	var issuerURL *url.URL
	var err error

	if config.Auth0Domain != "" {
		issuerURL, err = url.Parse("https://" + config.Auth0Domain + "/")
	} else {
		issuerURL, err = url.Parse(config.IssuerURL)
	}
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{config.Auth0Audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		// copied from auth0's DefaultErrorHandler, but with some extra logging and reporting
		w.Header().Set("Content-Type", "application/json")
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(
			attribute.String("ovm.auth.error", err.Error()),
			attribute.String("ovm.auth.audience", config.Auth0Audience),
			attribute.String("ovm.auth.domain", config.Auth0Domain),
			attribute.String("ovm.auth.expectedIssuer", issuerURL.String()),
		)

		switch {
		case errors.Is(err, jwtmiddleware.ErrJWTMissing):
			// since connectrpc would translate the original `BadRequest` to a
			// `CodeInternal` instead of something sensible, we also need to
			// return StatusUnauthorized here, to provide the correct status
			// code to the client.
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"JWT is missing."}`))
		case errors.Is(err, jwtmiddleware.ErrJWTInvalid):
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"JWT is invalid."}`))
		default:
			span.SetStatus(codes.Error, "Something went wrong while checking the JWT")
			sentry.CaptureException(err)

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"Something went wrong while checking the JWT."}`))
		}
	}

	// Set up token extractors based on what env vars are available
	extractors := []jwtmiddleware.TokenExtractor{
		jwtmiddleware.AuthHeaderTokenExtractor,
	}

	for _, cookieName := range config.AuthCookieNames {
		extractors = append(extractors, jwtmiddleware.CookieTokenExtractor(cookieName))
	}

	tokenExtractor := jwtmiddleware.MultiTokenExtractor(extractors...)

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
		jwtmiddleware.WithTokenExtractor(tokenExtractor),
	)

	jwtValidationMiddleware := middleware.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract account name and setup otel attributes after the JWT was validated, but before the actual handler runs
		claims := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

		token, err := tokenExtractor(r)
		// we should never hit this as the middleware wouldn't call the handler
		if err != nil {
			// This is not ErrJWTMissing because an error here means that the
			// tokenExtractor had an error and _not_ that the token was missing.
			errorHandler(w, r, fmt.Errorf("error extracting token: %w", err))
			return
		}

		customClaims := claims.CustomClaims.(*CustomClaims)
		if customClaims == nil {
			errorHandler(w, r, fmt.Errorf("couldn't get claims from: %v", claims))
			return
		}

		ctx := r.Context()

		// note that the values are looked up in last-in-first-out order, so
		// there is an absolutely minor perf optimisation to have the context
		// values set in ascending order of access frequency.
		ctx = context.WithValue(ctx, UserTokenContextKey{}, token)
		ctx = context.WithValue(ctx, CustomClaimsContextKey{}, customClaims)
		ctx = context.WithValue(ctx, CurrentSubjectContextKey{}, claims.RegisteredClaims.Subject)
		ctx = context.WithValue(ctx, AccountNameContextKey{}, customClaims.AccountName)

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.auth.accountName", customClaims.AccountName),
			attribute.Int64("ovm.auth.expiry", claims.RegisteredClaims.Expiry),
			attribute.String("ovm.auth.scopes", customClaims.Scope),
			// subject is the auth0 client id or user id
			attribute.String("ovm.auth.subject", claims.RegisteredClaims.Subject),
		)

		// if its a service impersonating an account, we should mark it as impersonation
		if strings.HasSuffix(claims.RegisteredClaims.Subject, "@clients") {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.Bool("ovm.auth.impersonation", true),
			)
		}

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	}))

	// Basically what I need to do here is I need to have a middleware that
	// checks for bypassing, then passes on to middleware.checkJWT.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

		var shouldBypass bool

		// If config.BypassAuth is true then bypass
		if config.BypassAuth {
			shouldBypass = true
		}

		// If we aren't bypassing always and we have a regex then check if we
		// should bypass
		if !shouldBypass && config.BypassAuthForPaths != nil {
			shouldBypass = config.BypassAuthForPaths.MatchString(r.URL.Path)
			if shouldBypass {
				span.SetAttributes(attribute.String("ovm.auth.bypassedPath", r.URL.Path))
			}
		}

		span.SetAttributes(attribute.Bool("ovm.auth.bypass", shouldBypass))

		if shouldBypass {
			ctx = OverrideAuth(ctx, WithBypassScopeCheck())

			r = r.Clone(ctx)

			// Call the next handler without adding any JWT validation
			next.ServeHTTP(w, r)
		} else {
			// Otherwise we need to inject the JWT validation middleware
			jwtValidationMiddleware.ServeHTTP(w, r)
		}
	})
}

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope       string `json:"scope"`
	AccountName string `json:"https://api.overmind.tech/account-name"`
}

// HasScope checks whether our claims have a specific scope.
func (c CustomClaims) HasScope(expectedScope string) bool {
	result := strings.Split(c.Scope, " ")
	for i := range result {
		if result[i] == expectedScope {
			return true
		}
	}

	return false
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}
