package tracing

import (
	"context"
	"fmt"
	"net/http"
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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	"golang.org/x/sync/errgroup"
)

// logrusOtelErrorHandler routes OpenTelemetry SDK errors through logrus so they
// appear in our structured log pipeline (and therefore in Honeycomb) instead of
// being silently written to Go's default logger.
type logrusOtelErrorHandler struct{}

func (logrusOtelErrorHandler) Handle(err error) {
	log.WithError(err).Warn("OpenTelemetry SDK error")
}

const instrumentationName = "github.com/overmindtech/workspace"

// the following vars will be set during the build using `ldflags`, eg:
//
//	go build -ldflags "-X github.com/overmindtech/cli/go/tracing.version=$VERSION" -o your-app
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
		switch viper.GetString("run-mode") {
		case "release":
			environment = "prod"
		case "test":
			environment = "dogfood"
		case "debug":
			environment = "local"
		default:
			// Fallback to dev for backward compatibility
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
	} else {
		// If no Honeycomb API key is provided, use the hardcoded OTLP collector
		// endpoint, which is provided by the otel-collector service in the otel
		// namespace. Since this a node-local service, it does not use TLS.
		opts = append(opts,
			otlptracehttp.WithEndpoint("otelcol-node-opentelemetry-collector.otel.svc.cluster.local:4318"),
			otlptracehttp.WithInsecure(),
		)
	}

	return InitTracer(component, opts...)
}

// batcherOpts are the shared BatchSpanProcessor options applied to every
// exporter. A large queue (8192, 4x the default 2048) reduces the chance of
// silent span drops during burst load. We intentionally avoid WithBlocking()
// because it causes test suites to hang when no collector is reachable (the
// common case in CI). The 60s export timeout aligns with the OTLP HTTP
// exporter's 1-minute retry budget.
var batcherOpts = []sdktrace.BatchSpanProcessorOption{
	sdktrace.WithMaxQueueSize(8192),
	sdktrace.WithExportTimeout(60 * time.Second),
}

func InitTracer(component string, opts ...otlptracehttp.Option) error {
	otel.SetErrorHandler(logrusOtelErrorHandler{})

	otlpExp, err := otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	healthExp, err := otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
	if err != nil {
		return fmt.Errorf("creating health OTLP trace exporter: %w", err)
	}

	overmindSampler := NewOvermindSampler()
	res := tracingResource(component)

	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(otlpExp, batcherOpts...),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(overmindSampler)),
	}
	if viper.GetBool("stdout-trace-dump") {
		stdoutExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return err
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(stdoutExp, batcherOpts...))
	}
	tp = sdktrace.NewTracerProvider(tracerOpts...)

	otel.SetTracerProvider(tp)

	healthTp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(healthExp, batcherOpts...),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

func ShutdownTracer(ctx context.Context) {
	defer sentry.Flush(5 * time.Second)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	var g errgroup.Group

	// Do not nil healthTp or tp here: concurrent callers (e.g. health check
	// probes via HealthCheckTracerProvider) would panic on the nil guard.
	// Calling Shutdown on an already-shutdown provider is a safe no-op.
	if healthTp != nil {
		localTp := healthTp
		g.Go(func() error {
			if err := localTp.ForceFlush(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("Error flushing health tracer provider")
			}
			if err := localTp.Shutdown(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("Error shutting down health tracer provider")
				return err
			}
			return nil
		})
	}

	if tp != nil {
		localTp := tp
		g.Go(func() error {
			if err := localTp.ForceFlush(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("Error flushing tracer provider")
			}
			if err := localTp.Shutdown(ctx); err != nil {
				log.WithContext(ctx).WithError(err).Error("Error shutting down tracer provider")
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.WithContext(ctx).WithError(err).Error("Error during tracer shutdown")
	}
	log.WithContext(ctx).Trace("tracing has shut down")
}

// SamplingRule defines a single sampling rule with a rate and matching function
type SamplingRule struct {
	SampleRate   int
	ShouldSample func(sdktrace.SamplingParameters) bool
}

// OvermindSampler is a unified sampler that evaluates multiple sampling rules in order
type OvermindSampler struct {
	rules        []SamplingRule
	ruleSamplers []sdktrace.Sampler
}

// NewOvermindSampler creates a new unified sampler with the default rules
func NewOvermindSampler() *OvermindSampler {
	rules := []SamplingRule{
		{
			SampleRate:   200,
			ShouldSample: UserAgentMatcher("ELB-HealthChecker/2.0", "kube-probe/1.27+"),
		},
	}

	// Pre-allocate samplers for each rule
	ruleSamplers := make([]sdktrace.Sampler, 0, len(rules))
	for _, rule := range rules {
		var sampler sdktrace.Sampler
		switch {
		case rule.SampleRate <= 0:
			sampler = sdktrace.NeverSample()
		case rule.SampleRate == 1:
			sampler = sdktrace.AlwaysSample()
		default:
			sampler = sdktrace.TraceIDRatioBased(1.0 / float64(rule.SampleRate))
		}
		ruleSamplers = append(ruleSamplers, sampler)
	}

	return &OvermindSampler{
		rules:        rules,
		ruleSamplers: ruleSamplers,
	}
}

// UserAgentMatcher returns a function that matches specific user agents
func UserAgentMatcher(userAgents ...string) func(sdktrace.SamplingParameters) bool {
	return func(parameters sdktrace.SamplingParameters) bool {
		for _, attr := range parameters.Attributes {
			if (attr.Key == "http.user_agent" || attr.Key == "user_agent.original") &&
				slices.Contains(userAgents, attr.Value.AsString()) {
				return true
			}
		}
		return false
	}
}

// ShouldSample evaluates rules in order and returns the first matching decision
func (o *OvermindSampler) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	for i, rule := range o.rules {
		if rule.ShouldSample(parameters) {
			// Use the pre-allocated sampler for this rule
			result := o.ruleSamplers[i].ShouldSample(parameters)
			if result.Decision == sdktrace.RecordAndSample {
				result.Attributes = append(result.Attributes,
					attribute.Int("SampleRate", rule.SampleRate))
			}
			return result
		}
	}

	// Default to AlwaysSample if no rules match
	return sdktrace.AlwaysSample().ShouldSample(parameters)
}

// Description returns information describing the Sampler
func (o *OvermindSampler) Description() string {
	return "Unified Overmind sampler combining multiple sampling strategies"
}

// Version returns the version baked into the binary at build time.
func Version() string {
	return version
}

// HTTPClient returns an HTTP client with OpenTelemetry instrumentation.
// This replaces the deprecated otelhttp.DefaultClient and should be used
// throughout the codebase for HTTP requests that need tracing.
func HTTPClient() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}
