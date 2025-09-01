package tracing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	_ "embed"

	"github.com/MrAlias/otel-schema-utils/schema"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/overmindtech/workspace"

// the following vars will be set during the build using `ldflags`, eg:
//
//	go build -ldflags "-X github.com/overmindtech/cli/tracing.version=$VERSION" -o your-app
//
// This allows caching to work for dev and removes the last `go generate`
// requirement from the build. If we were embedding the version here each time
// we would always produce a slightly different compiled binary, and therefore
// it would look like there was a change each time
var (
	version = "dev"
	commit  = "none"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version),
		trace.WithInstrumentationAttributes(
			attribute.String("build.commit", commit),
		),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func Tracer() trace.Tracer {
	return tracer
}

// hasGitDir returns true if the current directory or any parent directory contains a .git directory
func hasGitDir() bool {
	// Start with the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return false
	}

	// Check the current directory and all parent directories
	for {
		// Check if .git exists in this directory
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			return true // Found a .git directory
		}

		// Get the parent directory
		parentDir := filepath.Dir(dir)

		// If we've reached the root directory, stop searching
		if parentDir == dir {
			break
		}

		// Move up to the parent directory
		dir = parentDir
	}

	return false // No .git directory found
}

func tracingResource(component string) *resource.Resource {
	// Identify your application using resource detection
	resources := []*resource.Resource{}

	// the EC2 detector takes ~10s to time out outside EC2
	// disable it if we're running from a git checkout
	if !hasGitDir() {
		ec2Res, err := resource.New(context.Background(), resource.WithDetectors(ec2.NewResourceDetector()))
		if err != nil {
			log.WithError(err).Error("error initialising EC2 resource detector")
			return nil
		}
		resources = append(resources, ec2Res)
	}

	// Needs https://github.com/open-telemetry/opentelemetry-go-contrib/issues/1856 fixed first
	// // the EKS detector is temperamental and doesn't like running outside of kube
	// // hence we need to keep it from running when we know there's no kube
	// if !viper.GetBool("disable-kube") {
	// 	// Use the AWS resource detector to detect information about the runtime environment
	// 	detectors = append(detectors, eks.NewResourceDetector())
	// }

	hostRes, err := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		log.WithError(err).Error("error initialising host resource")
		return nil
	}
	resources = append(resources, hostRes)

	localRes, err := resource.New(context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNameKey.String(component),
			semconv.ServiceVersionKey.String(version),
			attribute.String("build.commit", commit),
		),
	)
	if err != nil {
		log.WithError(err).Error("error initialising local resource")
		return nil
	}
	resources = append(resources, localRes)

	conv := schema.NewConverter(schema.DefaultClient)
	res, err := conv.MergeResources(context.Background(), semconv.SchemaURL, resources...)

	if err != nil {
		log.WithError(err).Error("error merging resource")
		return nil
	}
	return res
}

var tp *sdktrace.TracerProvider
var healthTp *sdktrace.TracerProvider

// HealthCheckTracerProvider returns the tracer provider used for health checks. This has a built-in 1:100 sampler for health checks that are not captured by the default UserAgentSampler for ELB and kube-probe requests.
func HealthCheckTracerProvider() *sdktrace.TracerProvider {
	if healthTp == nil {
		panic("tracer providers not initialised")
	}
	return healthTp
}

// healthCheckTracer is the tracer used for health checks. This is heavily sampled to avoid getting spammed by k8s or ELBs
func HealthCheckTracer() trace.Tracer {
	return HealthCheckTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version),
		trace.WithSchemaURL(semconv.SchemaURL),
		trace.WithInstrumentationAttributes(
			attribute.Bool("ovm.healthCheck", true),
		),
	)
}

// InitTracerWithUpstreams initialises the tracer with uploading directly to Honeycomb and sentry if `honeycombApiKey` and `sentryDSN` is set respectively. `component` is used as the service name.
func InitTracerWithUpstreams(component, honeycombApiKey, sentryDSN string, opts ...otlptracehttp.Option) error {
	if sentryDSN != "" {
		var environment string
		if viper.GetString("run-mode") == "release" {
			environment = "prod"
		} else {
			environment = "dev"
		}
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
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
		log.Trace("sentry configured")
	}

	if honeycombApiKey != "" {
		opts = append(opts,
			otlptracehttp.WithEndpoint("api.honeycomb.io"),
			otlptracehttp.WithHeaders(map[string]string{"x-honeycomb-team": honeycombApiKey}),
		)
	}

	return InitTracer(component, opts...)
}

func InitTracer(component string, opts ...otlptracehttp.Option) error {
	client := otlptracehttp.NewClient(opts...)
	otlpExp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(otlpExp),
		sdktrace.WithResource(tracingResource(component)),
		sdktrace.WithSampler(sdktrace.ParentBased(NewUserAgentSampler(200, "ELB-HealthChecker/2.0", "kube-probe/1.27+"))),
	}
	if viper.GetBool("stdout-trace-dump") {
		stdoutExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return err
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(stdoutExp))
	}
	tp = sdktrace.NewTracerProvider(tracerOpts...)

	tracerOpts = append(tracerOpts, sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased((0.1)))))
	healthTp = sdktrace.NewTracerProvider(tracerOpts...)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

func ShutdownTracer(ctx context.Context) {
	// Flush buffered events before the program terminates.
	defer sentry.Flush(5 * time.Second)

	// detach from the parent's cancellation, and ensure that we do not wait
	// indefinitely on the trace provider shutdown
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
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
	userAgents          []string
	innerSampler        sdktrace.Sampler
	sampleRateAttribute attribute.KeyValue
}

func NewUserAgentSampler(sampleRate int, userAgents ...string) *UserAgentSampler {
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
		userAgents:          userAgents,
		innerSampler:        innerSampler,
		sampleRateAttribute: attribute.Int("SampleRate", sampleRate),
	}
}

// ShouldSample returns a SamplingResult based on a decision made from the
// passed parameters.
func (h *UserAgentSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	for _, attr := range parameters.Attributes {
		if (attr.Key == "http.user_agent" || attr.Key == "user_agent.original") && slices.Contains(h.userAgents, attr.Value.AsString()) {
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

// Version returns the version baked into the binary at build time.
func Version() string {
	return version
}
