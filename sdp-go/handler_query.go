// Code generated by "genhandler Query"; DO NOT EDIT

package sdp

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/tracing"
	"go.opentelemetry.io/otel/trace"
)

func NewQueryHandler(spanName string, h func(ctx context.Context, i *Query), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i Query
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, &i)
		},
		tracing.Tracer(),
	)
}

func NewRawQueryHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *Query), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i Query
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, m, &i)
		},
		tracing.Tracer(),
	)
}

func NewAsyncRawQueryHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *Query), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewAsyncOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i Query
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, m, &i)
		},
		tracing.Tracer(),
	)
}
