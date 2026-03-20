package sdpcache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// lookupBackend is the storage-facing interface used by lookupCoordinator.
// Implementations should focus on cache I/O while lookupCoordinator owns
// pending-work deduplication and shared branching behavior.
type lookupBackend interface {
	Search(ctx context.Context, ck CacheKey) ([]*sdp.Item, error)
	Delete(ck CacheKey)
}

// lookupCoordinator centralizes shared Lookup control flow:
// cache miss deduplication, wait/re-check behavior, error classification,
// and GET cardinality validation.
type lookupCoordinator struct {
	pending *pendingWork
}

func newLookupCoordinator(pending *pendingWork) *lookupCoordinator {
	if pending == nil {
		pending = newPendingWork()
	}

	return &lookupCoordinator{
		pending: pending,
	}
}

func (lc *lookupCoordinator) doneForMiss(ck CacheKey) func() {
	if lc == nil || lc.pending == nil {
		return noopDone
	}

	key := ck.String()
	var once sync.Once

	return func() {
		once.Do(func() {
			lc.pending.Complete(key)
		})
	}
}

func (lc *lookupCoordinator) Lookup(
	ctx context.Context,
	backend lookupBackend,
	ck CacheKey,
	requestedMethod sdp.QueryMethod,
) (bool, []*sdp.Item, *sdp.QueryError, func()) {
	span := trace.SpanFromContext(ctx)

	initialSearchStart := time.Now()
	items, err := backend.Search(ctx, ck)
	span.SetAttributes(attribute.Float64("ovm.cache.initialSearchDurationMs", float64(time.Since(initialSearchStart).Milliseconds())))

	if err != nil {
		var qErr *sdp.QueryError

		if errors.Is(err, ErrCacheNotFound) {
			shouldWork, entry := lc.pending.StartWork(ck.String())
			if shouldWork {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "cache miss"),
					attribute.Bool("ovm.cache.hit", false),
					attribute.Bool("ovm.cache.workPending", false),
				)
				return false, nil, nil, lc.doneForMiss(ck)
			}

			pendingWaitStart := time.Now()
			ok := lc.pending.Wait(ctx, entry)
			pendingWaitDuration := time.Since(pendingWaitStart)
			span.SetAttributes(
				attribute.Float64("ovm.cache.pendingWaitDurationMs", float64(pendingWaitDuration.Milliseconds())),
				attribute.Bool("ovm.cache.pendingWaitSuccess", ok),
			)

			if !ok {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "pending work cancelled or timeout"),
					attribute.Bool("ovm.cache.hit", false),
				)
				return false, nil, nil, noopDone
			}

			recheckStart := time.Now()
			items, recheckErr := backend.Search(ctx, ck)
			span.SetAttributes(attribute.Float64("ovm.cache.recheckSearchDurationMs", float64(time.Since(recheckStart).Milliseconds())))
			if recheckErr != nil {
				if errors.Is(recheckErr, ErrCacheNotFound) {
					span.SetAttributes(
						attribute.String("ovm.cache.result", "pending work completed but cache still empty"),
						attribute.Bool("ovm.cache.hit", false),
					)
					return false, nil, nil, noopDone
				}

				var recheckQErr *sdp.QueryError
				if errors.As(recheckErr, &recheckQErr) {
					span.SetAttributes(
						attribute.String("ovm.cache.result", "cache hit from pending work: error"),
						attribute.Bool("ovm.cache.hit", true),
					)
					return true, nil, recheckQErr, noopDone
				}

				span.SetAttributes(
					attribute.String("ovm.cache.result", "unexpected error on re-check"),
					attribute.Bool("ovm.cache.hit", false),
				)
				return false, nil, nil, noopDone
			}

			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit from pending work"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, items, nil, noopDone
		}

		if errors.As(err, &qErr) {
			if qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				span.SetAttributes(attribute.String("ovm.cache.result", "cache hit: item not found"))
			} else {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "cache hit: QueryError"),
					attribute.String("ovm.cache.error", err.Error()),
				)
			}

			span.SetAttributes(attribute.Bool("ovm.cache.hit", true))
			return true, nil, qErr, noopDone
		}

		qErr = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       ck.SST.Scope,
			SourceName:  ck.SST.SourceName,
			ItemType:    ck.SST.Type,
		}

		span.SetAttributes(
			attribute.String("ovm.cache.error", err.Error()),
			attribute.String("ovm.cache.result", "cache hit: unknown QueryError"),
			attribute.Bool("ovm.cache.hit", true),
		)
		return true, nil, qErr, noopDone
	}

	if requestedMethod == sdp.QueryMethod_GET {
		if len(items) < 2 {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit: 1 item"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, items, nil, noopDone
		}

		span.SetAttributes(
			attribute.String("ovm.cache.result", "cache returned >1 value, purging and continuing"),
			attribute.Int("ovm.cache.numItems", len(items)),
			attribute.Bool("ovm.cache.hit", false),
		)
		backend.Delete(ck)
		return false, nil, nil, noopDone
	}

	span.SetAttributes(
		attribute.String("ovm.cache.result", "cache hit: multiple items"),
		attribute.Int("ovm.cache.numItems", len(items)),
		attribute.Bool("ovm.cache.hit", true),
	)
	return true, items, nil, noopDone
}
