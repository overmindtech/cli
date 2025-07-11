package sdp

import (
	"context"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type CtxMsgHandler func(ctx context.Context, msg *nats.Msg)

func NewOtelExtractingHandler(spanName string, h CtxMsgHandler, t trace.Tracer, spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	if h == nil {
		return nil
	}

	return func(msg *nats.Msg) {
		ctx := context.Background()

		ctx = otel.GetTextMapPropagator().Extract(ctx, tracing.NewNatsHeaderCarrier(msg.Header))

		// don't start a span when we have no spanName
		if spanName != "" {
			var span trace.Span
			ctx, span = t.Start(ctx, spanName, spanOpts...)
			defer span.End()
		}

		h(ctx, msg)
	}
}

func NewAsyncOtelExtractingHandler(spanName string, h CtxMsgHandler, t trace.Tracer, spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	if h == nil {
		return nil
	}

	return func(msg *nats.Msg) {
		go func() {
			defer sentry.Recover()

			ctx := context.Background()
			ctx = otel.GetTextMapPropagator().Extract(ctx, tracing.NewNatsHeaderCarrier(msg.Header))

			// don't start a span when we have no spanName
			if spanName != "" {
				var span trace.Span
				ctx, span = t.Start(ctx, spanName, spanOpts...)
				defer span.End()
			}

			h(ctx, msg)
		}()
	}
}

func InjectOtelTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}

	otel.GetTextMapPropagator().Inject(ctx, tracing.NewNatsHeaderCarrier(msg.Header))
}

type sentryInterceptor struct{}

// NewSentryInterceptor pass this to connect handlers as `connect.WithInterceptors(NewSentryInterceptor())` to recover from panics in the handler and report them to sentry. Otherwise panics get recovered by connect-go itself and do not get reported to sentry.
func NewSentryInterceptor() connect.Interceptor {
	return &sentryInterceptor{}
}

func (i *sentryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	// Same as previous UnaryInterceptorFunc.
	return connect.UnaryFunc(func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		defer sentry.Recover()
		return next(ctx, req)
	})
}

func (*sentryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		defer sentry.Recover()
		conn := next(ctx, spec)
		return conn
	})
}

func (i *sentryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		defer sentry.Recover()
		return next(ctx, conn)
	})
}
