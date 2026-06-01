package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupHealthProbeTracing(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	})

	return exp
}

func assertNoLegacyHealthcheckSpans(t *testing.T, spans tracetest.SpanStubs) {
	t.Helper()

	for _, s := range spans {
		if s.Name == "healthcheck.liveness" || s.Name == "healthcheck.readiness" {
			t.Errorf("unexpected legacy healthcheck span name %q", s.Name)
		}
	}
}

func findHTTPProbeSpan(t *testing.T, spans tracetest.SpanStubs, wantRoute string) tracetest.SpanStub {
	t.Helper()

	for _, s := range spans {
		if s.Name != "GET "+wantRoute {
			continue
		}
		for _, a := range s.Attributes {
			if string(a.Key) == "http.route" && a.Value.AsString() == wantRoute {
				return s
			}
		}
	}
	t.Fatalf("expected span GET %s with http.route=%s, got spans: %v", wantRoute, wantRoute, spanNames(spans))
	return tracetest.SpanStub{}
}

func spanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, 0, len(spans))
	for _, s := range spans {
		names = append(names, s.Name)
	}
	return names
}

func attrString(span tracetest.SpanStub, key string) (string, bool) {
	for _, a := range span.Attributes {
		if string(a.Key) == key {
			return a.Value.AsString(), true
		}
	}
	return "", false
}

func TestHealthProbeHandler_ReadinessUninitialized(t *testing.T) {
	exp := setupHealthProbeTracing(t)

	ec := EngineConfig{SourceName: "test-source"}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	handler := e.healthProbeHandler()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	spans := exp.GetSpans()
	assertNoLegacyHealthcheckSpans(t, spans)

	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span, got %d: %v", len(spans), spanNames(spans))
	}

	span := findHTTPProbeSpan(t, spans, "/healthz/ready")
	healthType, ok := attrString(span, "ovm.healthcheck.type")
	if !ok || healthType != "readiness" {
		t.Errorf("expected ovm.healthcheck.type=readiness, got %q (ok=%v)", healthType, ok)
	}

	statusOK := false
	for _, a := range span.Attributes {
		if string(a.Key) == "http.response.status_code" && a.Value.AsInt64() == int64(http.StatusServiceUnavailable) {
			statusOK = true
		}
	}
	if !statusOK {
		t.Errorf("expected http.response.status_code=%d on probe span", http.StatusServiceUnavailable)
	}
}

func TestHealthProbeHandler_ReadinessOK(t *testing.T) {
	exp := setupHealthProbeTracing(t)

	ec := EngineConfig{SourceName: "test-source"}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	e.MarkAdaptersInitialized()

	handler := e.healthProbeHandler()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	spans := exp.GetSpans()
	assertNoLegacyHealthcheckSpans(t, spans)

	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span, got %d: %v", len(spans), spanNames(spans))
	}

	span := findHTTPProbeSpan(t, spans, "/healthz/ready")
	healthType, ok := attrString(span, "ovm.healthcheck.type")
	if !ok || healthType != "readiness" {
		t.Errorf("expected ovm.healthcheck.type=readiness, got %q (ok=%v)", healthType, ok)
	}
}

func TestHealthProbeHandler_LivenessNoNATS(t *testing.T) {
	exp := setupHealthProbeTracing(t)

	ec := EngineConfig{SourceName: "test-source"}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	handler := e.healthProbeHandler()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz/alive", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	spans := exp.GetSpans()
	assertNoLegacyHealthcheckSpans(t, spans)

	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span, got %d: %v", len(spans), spanNames(spans))
	}

	span := findHTTPProbeSpan(t, spans, "/healthz/alive")
	healthType, ok := attrString(span, "ovm.healthcheck.type")
	if !ok || healthType != "liveness" {
		t.Errorf("expected ovm.healthcheck.type=liveness, got %q (ok=%v)", healthType, ok)
	}

	statusOK := false
	for _, a := range span.Attributes {
		if string(a.Key) == "http.response.status_code" && a.Value.AsInt64() == int64(http.StatusServiceUnavailable) {
			statusOK = true
		}
	}
	if !statusOK {
		t.Errorf("expected http.response.status_code=%d on probe span", http.StatusServiceUnavailable)
	}
}

