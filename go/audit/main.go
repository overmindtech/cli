package audit

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type contextKey struct{}

// AuditData holds identity fields populated by auth middleware for
// post-request audit logging. The audit middleware places a mutable
// *AuditData in the request context before calling inner handlers;
// auth fills it after token validation so the log emitted after
// the response contains the correct identity.
type AuditData struct {
	Subject     string
	AccountName string
	Scopes      string
}

// AuditDataFromContext returns the AuditData pointer placed in context
// by the audit middleware. Returns nil when called outside the chain.
func AuditDataFromContext(ctx context.Context) *AuditData {
	ad, _ := ctx.Value(contextKey{}).(*AuditData)
	return ad
}

// Option configures the audit middleware.
type Option func(*auditConfig)

type auditConfig struct {
	excludePaths map[string]bool
}

// WithExcludePaths skips audit logging for the given exact request
// paths (e.g. "/healthz").
func WithExcludePaths(paths ...string) Option {
	return func(c *auditConfig) {
		for _, p := range paths {
			c.excludePaths[p] = true
		}
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wroteHeader {
		sr.status = code
		sr.wroteHeader = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if !sr.wroteHeader {
		sr.WriteHeader(http.StatusOK)
	}
	return sr.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter, preserving optional
// interfaces (Flusher, Hijacker, etc.) for http.ResponseController.
func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

// Hijack implements http.Hijacker by delegating to the underlying
// ResponseWriter. This is required for WebSocket upgrade handshakes
// which do direct type assertions on the writer.
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := sr.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter does not support hijacking")
}

// Flush implements http.Flusher by delegating to the underlying
// ResponseWriter. This is needed for streaming responses (SSE, etc.).
func (sr *statusRecorder) Flush() {
	if f, ok := sr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// NewAuditMiddleware returns middleware that emits a structured audit
// log entry after each request completes. Identity fields (sub, account,
// scopes) are populated by auth middleware via [AuditDataFromContext].
//
// The middleware must wrap the handler chain from outside otelhttp so
// that audit logs are not exported to the tracing backend.
func NewAuditMiddleware(logger *log.Logger, opts ...Option) func(next http.Handler) http.Handler {
	cfg := &auditConfig{excludePaths: make(map[string]bool)}
	for _, o := range opts {
		o(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.excludePaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			ad := &AuditData{}
			ctx := context.WithValue(r.Context(), contextKey{}, ad)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(ctx))

			logger.WithContext(ctx).
				WithField("method", r.Method).
				WithField("url", r.URL.String()).
				WithField("status", rec.status).
				WithField("sub", ad.Subject).
				WithField("account", ad.AccountName).
				WithField("ovm.audit", true).
				WithField("scopes", ad.Scopes).
				Info("audit")
		})
	}
}
