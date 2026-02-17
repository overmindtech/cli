package sdp

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceContextPropagation(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tc := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	outerCtx := context.Background()
	outerCtx, outerSpan := tp.Tracer("outerTracer").Start(outerCtx, "outer span")
	defer outerSpan.End()
	// outerJson, err := outerSpan.SpanContext().MarshalJSON()
	// if err != nil {
	// 	t.Errorf("error marshalling outerSpan: %v", err)
	// } else {
	// 	if !bytes.Equal(outerJson, []byte("{\"TraceID\":\"00000000000000000000000000000000\",\"SpanID\":\"0000000000000000\",\"TraceFlags\":\"00\",\"TraceState\":\"\",\"Remote\":false}")) {
	// 		t.Errorf("outer span has unexpected context: %v", string(outerJson))
	// 	}
	// }
	handlerCalled := make(chan struct{})
	_, err := tc.Subscribe("test.subject", NewOtelExtractingHandler("inner span", func(innerCtx context.Context, msg *nats.Msg) {
		_, innerSpan := tp.Tracer("innerTracer").Start(innerCtx, "innerSpan")
		// innerJson, err := innerSpan.SpanContext().MarshalJSON()
		// if err != nil {
		// 	t.Errorf("error marshalling innerSpan: %v", err)
		// } else {
		// 	if !bytes.Equal(innerJson, []byte("{\"TraceID\":\"00000000000000000000000000000000\",\"SpanID\":\"0000000000000000\",\"TraceFlags\":\"00\",\"TraceState\":\"\",\"Remote\":false}")) {
		// 		t.Errorf("inner span has unexpected context: %v", string(innerJson))
		// 	}
		// }
		if innerSpan.SpanContext().TraceID() != outerSpan.SpanContext().TraceID() {
			t.Error("inner span did not link up to outer span")
		}

		// clean up
		innerSpan.End()

		// finish the test
		handlerCalled <- struct{}{}
	}, tp.Tracer("providedTracer")))
	if err != nil {
		t.Errorf("error subscribing: %v", err)
	}

	m := &nats.Msg{
		Subject: "test.subject",
		Data:    make([]byte, 0),
	}

	go func() {
		InjectOtelTraceContext(outerCtx, m)
		err = tc.PublishMsg(outerCtx, m)
		if err != nil {
			t.Errorf("error publishing message: %v", err)
		}
	}()

	// Wait for the handler to be called
	<-handlerCalled
}
