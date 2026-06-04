package discovery

import (
	"net/http"
	"strings"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// wrapHealthProbeHandler applies the source health-probe middleware chain.
//
// Chain (outer → inner):
//  1. sentryhttp — per-request panic capture with Repanic: true
//  2. otelhttp — creates the HTTP span; span name prefers http.Request.Pattern
//     (set by http.ServeMux in Go 1.22+) over the fallback service name
//  3. routeAttributeMiddleware — sets http.route after the mux runs
//  4. albTraceIDMiddleware — records X-Amzn-Trace-Id as aws.alb.trace_id
//
// This is intentionally a local copy of the subset of go/startup.WrapHandler
// that source health probes use. go/discovery is synced into the public
// overmindtech/cli repo via copybara, but go/startup is not — see
// copy.bara.sky. Importing go/startup from here breaks `go mod tidy` in cli
// because the import path becomes unresolvable in the synced module.
//
// Keep behaviour in sync with go/startup.WrapHandler when called with only
// WithServiceName (no audit logger).
func wrapHealthProbeHandler(handler http.Handler, serviceName string) http.Handler {
	h := albTraceIDMiddleware(handler)
	h = routeAttributeMiddleware(h)

	h = otelhttp.NewHandler(
		h, serviceName,
		otelhttp.WithSpanNameFormatter(patternSpanNameFormatter),
	)

	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})
	h = sentryHandler.Handle(h)

	return h
}

// patternSpanNameFormatter returns the http.Request.Pattern (set by
// http.ServeMux in Go 1.22+) when available, falling back to the static
// operation name for unmatched routes (404s).
func patternSpanNameFormatter(operation string, r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return operation
}

// routeAttributeMiddleware sets the http.route span attribute after the
// inner handler runs, using http.Request.Pattern populated by http.ServeMux.
// otelhttp's RequestTraceAttrs runs before the mux, so Pattern is still empty
// at that point — this middleware fills the gap.
func routeAttributeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if r.Pattern != "" {
			route := r.Pattern
			if idx := strings.IndexByte(route, '/'); idx >= 0 {
				route = route[idx:]
			}
			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("http.route", route),
			)
		}
	})
}

// albTraceIDMiddleware extracts the AWS ALB trace ID from the
// X-Amzn-Trace-Id header and records it as a span attribute.
func albTraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Amzn-Trace-Id"); v != "" {
			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("aws.alb.trace_id", v),
			)
		}
		next.ServeHTTP(w, r)
	})
}
