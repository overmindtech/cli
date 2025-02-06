package tracing

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LogRecoverToReturn Recovers from a panic, logs and forwards it sentry and otel, then returns
// Does nothing when there is no panic.
func LogRecoverToReturn(ctx context.Context, loc string) {
	err := recover()
	if err == nil {
		return
	}

	stack := string(debug.Stack())
	HandleError(ctx, loc, err, stack)
}

// LogRecoverToExit Recovers from a panic, logs and forwards it sentry and otel, then exits
// Does nothing when there is no panic.
func LogRecoverToExit(ctx context.Context, loc string) {
	err := recover()
	if err == nil {
		return
	}

	stack := string(debug.Stack())
	HandleError(ctx, loc, err, stack)

	// ensure that errors still get sent out
	ShutdownTracer(ctx)

	os.Exit(1)
}

func HandleError(ctx context.Context, loc string, err interface{}, stack string) {
	msg := fmt.Sprintf("unhandled panic in %v, exiting: %v", loc, err)

	hub := sentry.CurrentHub()
	if hub != nil {
		hub.Recover(err)
	}

	// always log to stderr (no WithContext!)
	log.WithFields(log.Fields{"loc": loc, "stack": stack}).Error(msg)

	// if we have a context, try attaching additional info to the span
	if ctx != nil {
		log.WithContext(ctx).WithFields(log.Fields{"loc": loc, "stack": stack}).Error(msg)
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("ovm.panic.loc", loc))
		span.SetAttributes(attribute.String("ovm.panic.stack", stack))
	}
}
