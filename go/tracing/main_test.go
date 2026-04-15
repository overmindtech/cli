package tracing

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracingResource(t *testing.T) {
	resource := tracingResource("test-component")
	if resource == nil {
		t.Error("Could not initialize tracing resource. Check the log!")
	}
}

func TestShutdownProvider(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()

	tp = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))

	if tp == nil {
		t.Fatal("expected tp to be non-nil after init")
	}

	ShutdownTracer(context.Background())

	// After shutdown, calling Shutdown again should be a safe no-op
	// (the SDK guards with stopOnce).
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("second tp.Shutdown should be a no-op, got: %v", err)
	}
}

func TestShutdownIdempotent(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()

	tp = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))

	ShutdownTracer(context.Background())
	ShutdownTracer(context.Background()) // must not panic
}

func TestErrorHandlerRegistered(t *testing.T) {
	otel.SetErrorHandler(logrusOtelErrorHandler{})

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	otel.Handle(fmt.Errorf("test SDK error"))

	if !bytes.Contains(buf.Bytes(), []byte("OpenTelemetry SDK error")) {
		t.Errorf("expected logrus to contain 'OpenTelemetry SDK error', got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("test SDK error")) {
		t.Errorf("expected logrus to contain the original error, got: %s", buf.String())
	}
}

func TestBatcherOptsQueueSize(t *testing.T) {
	found := false
	for _, opt := range batcherOpts {
		// Apply each option to a zero-value struct and check the result.
		var o sdktrace.BatchSpanProcessorOptions
		opt(&o)
		if o.MaxQueueSize == 8192 {
			found = true
		}
	}
	if !found {
		t.Error("batcherOpts should set MaxQueueSize to 8192")
	}
}

func TestInitTracerSetsErrorHandler(t *testing.T) {
	// Use a deliberately broken endpoint so the exporter creation succeeds
	// but no actual spans are shipped.
	err := InitTracer("test-component",
		otlptracehttp.WithEndpoint("localhost:0"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	t.Cleanup(func() { ShutdownTracer(context.Background()) })

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	otel.Handle(fmt.Errorf("custom test error"))

	if !bytes.Contains(buf.Bytes(), []byte("OpenTelemetry SDK error")) {
		t.Errorf("after InitTracer, OTel errors should be routed to logrus; got: %s", buf.String())
	}
}

func TestHTTPClient_ProducesSpans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	exp := tracetest.NewInMemoryExporter()
	testTP := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { _ = testTP.Shutdown(context.Background()) })

	origTP := otel.GetTracerProvider()
	otel.SetTracerProvider(testTP)
	t.Cleanup(func() { otel.SetTracerProvider(origTP) })

	client := HTTPClient()

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test-path", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	_ = testTP.ForceFlush(ctx)
	spans := exp.GetSpans()

	if len(spans) == 0 {
		t.Fatal("expected at least one span from HTTPClient(), got 0")
	}

	var found bool
	for _, s := range spans {
		if s.SpanKind.String() == "client" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(spans))
		for i, s := range spans {
			names[i] = fmt.Sprintf("%s (kind=%s)", s.Name, s.SpanKind)
		}
		t.Fatalf("no client span found; spans: %v", names)
	}
}
