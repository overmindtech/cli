package tracing

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
)

//go:generate sh -c "echo -n $(git describe --tags --long) > commit.txt"
//go:embed commit.txt
var instrumentationVersion string

const instrumentationName = "github.com/overmindtech/cli"

var tracer = otel.GetTracerProvider().Tracer(
	instrumentationName,
	trace.WithInstrumentationVersion(instrumentationVersion),
	trace.WithSchemaURL(semconv.SchemaURL),
)

func Tracer() trace.Tracer {
	return tracer
}

func tracingResource() *resource.Resource {
	// Identify your application using resource detection
	detectors := []resource.Detector{}

	res, err := resource.New(context.Background(),
		resource.WithDetectors(detectors...),
		// replace the default detectors
		resource.WithHost(),
		resource.WithOS(),
		// resource.WithProcess(), // don't capture potentially sensitive customer info
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
		resource.WithSchemaURL(semconv.SchemaURL),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNameKey.String("overmind-cli"),
			semconv.ServiceVersionKey.String(instrumentationVersion),
		),
	)
	if err != nil {
		log.Errorf("resource.New: %v", err)
		return nil
	}
	return res
}

var tp *sdktrace.TracerProvider

func InitTracerWithHoneycomb(key string, opts ...otlptracehttp.Option) error {
	if key != "" {
		opts = append(opts,
			otlptracehttp.WithEndpoint("api.honeycomb.io"),
			otlptracehttp.WithHeaders(map[string]string{"x-honeycomb-team": key}),
		)
	}
	return InitTracer(opts...)
}

func InitTracer(opts ...otlptracehttp.Option) error {
	if sentry_dsn := viper.GetString("sentry-dsn"); sentry_dsn != "" {
		var environment string
		if viper.GetString("run-mode") == "release" {
			environment = "prod"
		} else {
			environment = "dev"
		}
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentry_dsn,
			AttachStacktrace: true,
			EnableTracing:    false,
			Environment:      environment,
			// Set TracesSampleRate to 1.0 to capture 100%
			// of transactions for performance monitoring.
			// We recommend adjusting this value in production,
			TracesSampleRate: 1.0,
		})
		if err != nil {
			log.Errorf("sentry.Init: %s", err)
		}
		// setup recovery for an unexpected panic in this function
		defer sentry.Flush(2 * time.Second)
		defer sentry.Recover()
		log.Info("sentry configured")
	}

	client := otlptracehttp.NewClient(opts...)
	otlpExp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(otlpExp, sdktrace.WithMaxQueueSize(50000)),
		sdktrace.WithResource(tracingResource()),
		sdktrace.WithSampler(sdktrace.ParentBased(NewUserAgentSampler("ELB-HealthChecker/2.0", 200))),
	}

	if viper.GetBool("stdout-trace-dump") {
		stdoutExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return err
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(stdoutExp))
	}
	tp = sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

// nolint: contextcheck // deliberate use of local context to avoid getting tangled up in any existing timeouts or cancels
func ShutdownTracer() {
	// Flush buffered events before the program terminates.
	defer sentry.Flush(5 * time.Second)

	// ensure that we do not wait indefinitely on the trace provider shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if tp != nil {
		if err := tp.ForceFlush(ctx); err != nil {
			log.WithContext(ctx).WithError(err).Error("Error flushing tracer provider")
		}
		if err := tp.Shutdown(ctx); err != nil {
			log.WithContext(ctx).WithError(err).Error("Error shutting down tracer provider")
		}
	}
	log.WithContext(ctx).Trace("tracing has shut down")
}

type UserAgentSampler struct {
	userAgent           string
	innerSampler        sdktrace.Sampler
	sampleRateAttribute attribute.KeyValue
}

func NewUserAgentSampler(userAgent string, sampleRate int) *UserAgentSampler {
	var innerSampler sdktrace.Sampler
	switch {
	case sampleRate <= 0:
		innerSampler = sdktrace.NeverSample()
	case sampleRate == 1:
		innerSampler = sdktrace.AlwaysSample()
	default:
		innerSampler = sdktrace.TraceIDRatioBased(1.0 / float64(sampleRate))
	}
	return &UserAgentSampler{
		userAgent:           userAgent,
		innerSampler:        innerSampler,
		sampleRateAttribute: attribute.Int("SampleRate", sampleRate),
	}
}

// ShouldSample returns a SamplingResult based on a decision made from the
// passed parameters.
func (h *UserAgentSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	for _, attr := range parameters.Attributes {
		if attr.Key == "http.user_agent" && attr.Value.AsString() == h.userAgent {
			result := h.innerSampler.ShouldSample(parameters)
			if result.Decision == sdktrace.RecordAndSample {
				result.Attributes = append(result.Attributes, h.sampleRateAttribute)
			}
			return result
		}
	}

	return sdktrace.AlwaysSample().ShouldSample(parameters)
}

// Description returns information describing the Sampler.
func (h *UserAgentSampler) Description() string {
	return "Simple Sampler based on the UserAgent of the request"
}

// LogRecoverToReturn Recovers from a panic, logs and forwards it sentry and otel, then returns
// Does nothing when there is no panic.
func LogRecoverToReturn(ctx context.Context, loc string) {
	err := recover()
	if err == nil {
		return
	}

	stack := string(debug.Stack())
	handleError(ctx, loc, err, stack)
}

// LogRecoverToExit Recovers from a panic, logs and forwards it sentry and otel, then exits
// Does nothing when there is no panic.
func LogRecoverToExit(ctx context.Context, loc string) {
	err := recover()
	if err == nil {
		return
	}

	stack := string(debug.Stack())
	handleError(ctx, loc, err, stack)

	// ensure that errors still get sent out
	ShutdownTracer()

	os.Exit(1)
}

func handleError(ctx context.Context, loc string, err interface{}, stack string) {
	msg := fmt.Sprintf("unhandled panic in %v, exiting: %v", loc, err)

	hub := sentry.CurrentHub()
	if hub != nil {
		hub.Recover(err)
	}

	if ctx != nil {
		log.WithContext(ctx).WithFields(log.Fields{"loc": loc, "stack": stack}).Error(msg)
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("ovm.panic.loc", loc))
		span.SetAttributes(attribute.String("ovm.panic.stack", stack))
	} else {
		log.WithFields(log.Fields{"loc": loc, "stack": stack}).Error(msg)
	}
}
