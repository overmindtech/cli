// Code generated by "genhandler UndoQuery"; DO NOT EDIT

package sdp

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
	"github.com/overmindtech/cli/tracing"
)

func NewUndoQueryHandler(spanName string, h func(ctx context.Context, i *UndoQuery), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i UndoQuery
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, &i)
		},
		tracing.Tracer(),
	)
}

func NewRawUndoQueryHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *UndoQuery), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i UndoQuery
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, m, &i)
		},
		tracing.Tracer(),
	)
}

func NewAsyncRawUndoQueryHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *UndoQuery), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewAsyncOtelExtractingHandler(
		spanName,
		func(ctx context.Context, m *nats.Msg) {
			var i UndoQuery
			err := Unmarshal(ctx, m.Data, &i)
			if err != nil {
				return
			}
			h(ctx, m, &i)
		},
		tracing.Tracer(),
	)
}
