package discovery

import (
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/overmindtech/cli/go/discovery/discovery"
	instrumentationVersion = "0.0.1"
)

// getTracer returns the discovery tracer from the current global TracerProvider.
// Call this at span creation time (not once at init) so tests can install an
// in-memory TracerProvider before running discovery code.
func getTracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
}
