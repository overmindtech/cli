package tracing

import (
	_ "embed"

	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/overmindtech/cli/sdp-go"

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func Tracer() trace.Tracer {
	return tracer
}
