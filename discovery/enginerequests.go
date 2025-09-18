package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewItemSubject Generates a random subject name for returning items e.g.
// return.item._INBOX.712ab421
func NewItemSubject() string {
	return fmt.Sprintf("return.item.%v", nats.NewInbox())
}

// NewResponseSubject Generates a random subject name for returning responses
// e.g. return.response._INBOX.978af6de
func NewResponseSubject() string {
	return fmt.Sprintf("return.response.%v", nats.NewInbox())
}

// HandleQuery Handles a single query. This includes responses, linking
// etc.
func (e *Engine) HandleQuery(ctx context.Context, query *sdp.Query) {
	var deadlineOverride bool

	// If there is no deadline OR further in the future than MaxRequestTimeout, clamp the deadline to MaxRequestTimeout
	maxRequestDeadline := time.Now().Add(e.MaxRequestTimeout)
	if query.GetDeadline() == nil || query.GetDeadline().AsTime().After(maxRequestDeadline) {
		query.Deadline = timestamppb.New(maxRequestDeadline)
		deadlineOverride = true
		log.WithContext(ctx).WithField("ovm.deadline", query.GetDeadline().AsTime()).Debug("capping deadline to MaxRequestTimeout")
	}

	// Add the query timeout to the context stack
	ctx, cancel := query.TimeoutContext(ctx)
	defer cancel()

	numExpandedQueries := len(e.sh.ExpandQuery(query))

	if numExpandedQueries == 0 {
		// If we don't have any relevant adapters, exit
		return
	}

	// Extract and parse the UUID
	u, uuidErr := uuid.FromBytes(query.GetUUID())

	// Only start the span if we actually have something that will respond
	ctx, span := tracer.Start(ctx, "HandleQuery", trace.WithAttributes(
		attribute.Int("ovm.discovery.numExpandedQueries", numExpandedQueries),
		attribute.String("ovm.sdp.uuid", u.String()),
		attribute.String("ovm.sdp.type", query.GetType()),
		attribute.String("ovm.sdp.method", query.GetMethod().String()),
		attribute.String("ovm.sdp.query", query.GetQuery()),
		attribute.String("ovm.sdp.scope", query.GetScope()),
		attribute.String("ovm.sdp.deadline", query.GetDeadline().AsTime().String()),
		attribute.Bool("ovm.sdp.deadlineOverridden", deadlineOverride),
		attribute.Bool("ovm.sdp.queryIgnoreCache", query.GetIgnoreCache()),
	))
	defer span.End()

	if query.GetRecursionBehaviour() != nil {
		span.SetAttributes(
			attribute.Int("ovm.sdp.linkDepth", int(query.GetRecursionBehaviour().GetLinkDepth())),
			attribute.Bool("ovm.sdp.followOnlyBlastPropagation", query.GetRecursionBehaviour().GetFollowOnlyBlastPropagation()),
		)
	}

	// Respond saying we've got it
	responder := sdp.ResponseSender{
		ResponseSubject: query.Subject(),
	}

	var pub sdp.EncodedConnection

	if e.IsNATSConnected() {
		span.SetAttributes(attribute.Bool("ovm.nats.connected", true))
		pub = e.natsConnection
	} else {
		span.SetAttributes(attribute.Bool("ovm.nats.connected", false))
		pub = NilConnection{}
	}

	ru := uuid.New()
	responder.Start(
		ctx,
		pub,
		e.EngineConfig.SourceName,
		ru,
	)

	qt := QueryTracker{
		Query:   query,
		Engine:  e,
		Context: ctx,
		Cancel:  cancel,
	}

	if uuidErr == nil {
		e.TrackQuery(u, &qt)
		defer e.DeleteTrackedQuery(u)
	}

	// the query tracker will send responses directly through the embedded
	// engine's nats connection
	_, _, _, err := qt.Execute(ctx)

	// If all failed then return an error
	if err != nil {
		if errors.Is(err, context.Canceled) {
			responder.CancelWithContext(ctx)
		} else {
			responder.ErrorWithContext(ctx)
		}

		span.SetAttributes(
			attribute.String("ovm.sdp.errorType", "OTHER"),
			attribute.String("ovm.sdp.errorString", err.Error()),
		)
	} else {
		responder.DoneWithContext(ctx)
	}
}

var listExecutionPoolCount atomic.Int32
var getExecutionPoolCount atomic.Int32

// ExecuteQuery Executes a single Query and returns the results without any
// linking. Will return an error if the Query couldn't be run.
//
// Items and errors will be sent to the supplied channels as they are found.
// Note that if these channels are not buffered, something will need to be
// receiving the results or this method will never finish. If results are not
// required the channels can be nil
func (e *Engine) ExecuteQuery(ctx context.Context, query *sdp.Query, responses chan<- *sdp.QueryResponse) error {
	span := trace.SpanFromContext(ctx)

	// Make sure we close channels once we're done
	if responses != nil {
		defer close(responses)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	expanded := e.sh.ExpandQuery(query)

	span.SetAttributes(
		attribute.Int("ovm.adapter.numExpandedQueries", len(expanded)),
	)

	if len(expanded) == 0 {
		responses <- sdp.NewQueryResponseFromError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "no matching adapters found",
			Scope:       query.GetScope(),
		})

		return errors.New("no matching adapters found")
	}

	// Since we need to wait for only the processing of this query's executions, we need a separate WaitGroup here
	// Overall MaxParallelExecutions evaluation is handled by e.executionPool
	wg := sync.WaitGroup{}
	expandedMutex := sync.RWMutex{}
	expandedMutex.RLock()
	for q, adapter := range expanded {
		wg.Add(1)
		// localize values for the closure below
		localQ, localAdapter := q, adapter

		var p *pool.Pool
		if localQ.GetMethod() == sdp.QueryMethod_LIST {
			p = e.listExecutionPool
			listExecutionPoolCount.Add(1)
		} else {
			p = e.getExecutionPool
			getExecutionPoolCount.Add(1)
		}

		// push all queued items through a goroutine to avoid blocking `ExecuteQuery` from progressing
		// as `executionPool.Go()` will block once the max parallelism is hit
		go func() {
			// queue everything into the execution pool
			defer tracing.LogRecoverToReturn(ctx, "ExecuteQuery outer")
			span.SetAttributes(
				attribute.Int("ovm.discovery.listExecutionPoolCount", int(listExecutionPoolCount.Load())),
				attribute.Int("ovm.discovery.getExecutionPoolCount", int(getExecutionPoolCount.Load())),
			)
			p.Go(func() {
				defer tracing.LogRecoverToReturn(ctx, "ExecuteQuery inner")
				defer func() {
					// Mark the work as done. This happens before we start
					// waiting on `expandedMutex` below, to ensure that the
					// queues can continue executing even if we are waiting on
					// the mutex.
					wg.Done()

					// Delete our query from the map so that we can track which
					// ones are still running
					expandedMutex.Lock()
					defer expandedMutex.Unlock()
					delete(expanded, localQ)
				}()
				defer func() {
					if localQ.GetMethod() == sdp.QueryMethod_LIST {
						listExecutionPoolCount.Add(-1)
					} else {
						getExecutionPoolCount.Add(-1)
					}
				}()

				// If the context is cancelled, don't even bother doing
				// anything. Since the `p.Go` will block, it's possible that if
				// the pool was exhausted, the context could be cancelled before
				// the goroutine is executed
				if ctx.Err() != nil {
					return
				}

				// Execute the query against the adapter
				e.Execute(ctx, localQ, localAdapter, responses)
			})
		}()
	}
	expandedMutex.RUnlock()

	waitGroupDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitGroupDone)
	}()

	select {
	case <-waitGroupDone:
		// All adapters have finished
	case <-ctx.Done():
		// The context was cancelled, this should have propagated to all the
		// adapters and therefore we should see the wait group finish very
		// quickly now. We will check this though to make sure. This will wait
		// until we reach Change Analysis SLO violation territory. If this is
		// too quick, we are only spamming logs for nothing.
		longRunningAdaptersTimeout := 2 * time.Minute

		// Wait for the wait group, but ping the logs if it's taking
		// too long
		func() {
			for {
				select {
				case <-waitGroupDone:
					return
				case <-time.After(longRunningAdaptersTimeout):
					// If we're here, then the wait group didn't finish in time
					expandedMutex.RLock()
					for q, adapter := range expanded {
						// There is a honeycomb trigger for this message:
						//
						// https://ui.honeycomb.io/overmind/environments/prod/datasets/kubernetes-metrics/triggers/saWNAnCAXNb
						//
						// This is to ensure we are aware of any adapters that
						// are taking too long to respond to a query, which
						// could indicate a bug in the adapter. Make sure to
						// keep the trigger and this message in sync.
						log.WithContext(ctx).WithFields(log.Fields{
							"ovm.query.uuid":    q.GetUUIDParsed().String(),
							"ovm.query.type":    q.GetType(),
							"ovm.query.scope":   q.GetScope(),
							"ovm.query.method":  q.GetMethod().String(),
							"ovm.query.adapter": adapter.Name(),
						}).Errorf("Wait group still running %v after context cancelled", longRunningAdaptersTimeout)
					}
					expandedMutex.RUnlock()
					// the query is already bolloxed up, we don't need continue to wait and spam the logs any more
					return
				}
			}
		}()
	}

	// If the context is cancelled, return that error
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

// Runs a query against an adapter. Returns an error if the query fails in a
// "fatal" way that should consider the query as failed. Other non-fatal errors
// should be sent on the stream. Channels for items and errors will NOT be
// closed by this function, the caller should do that as this will likely be
// called in parallel with other queries and the results should be merged
func (e *Engine) Execute(ctx context.Context, q *sdp.Query, adapter Adapter, responses chan<- *sdp.QueryResponse) {
	ctx, span := tracer.Start(ctx, "Execute", trace.WithAttributes(
		attribute.String("ovm.adapter.queryMethod", q.GetMethod().String()),
		attribute.String("ovm.adapter.queryType", q.GetType()),
		attribute.String("ovm.adapter.queryScope", q.GetScope()),
		attribute.String("ovm.adapter.name", adapter.Name()),
		attribute.String("ovm.adapter.query", q.GetQuery()),
	))
	defer span.End()

	// We want to avoid having a Get and a List running at the same time, we'd
	// rather run the List first, populate the cache, then have the Get just
	// grab the value from the cache. To this end we use a GetListMutex to allow
	// a List to block all subsequent Get queries until it is done
	switch q.GetMethod() {
	case sdp.QueryMethod_GET:
		e.gfm.GetLock(q.GetScope(), q.GetType())
		defer e.gfm.GetUnlock(q.GetScope(), q.GetType())
	case sdp.QueryMethod_LIST:
		e.gfm.ListLock(q.GetScope(), q.GetType())
		defer e.gfm.ListUnlock(q.GetScope(), q.GetType())
	case sdp.QueryMethod_SEARCH:
		// We don't need to lock for a search since they are independent and
		// will only ever have a cache hit if the query is identical
	}

	span.SetAttributes(
		attribute.String("ovm.adapter.queryType", q.GetType()),
		attribute.String("ovm.adapter.queryScope", q.GetScope()),
	)

	// Ensure that the span is closed when the context is done. This is based on
	// the assumption that some adapters may not respect the context deadline and
	// may run indefinitely. This ensures that we at least get notified about
	// it.
	go func() {
		<-ctx.Done()
		if ctx.Err() != nil {
			// get a fresh copy of the span to avoid data races
			span := trace.SpanFromContext(ctx)
			span.RecordError(ctx.Err())
			span.SetAttributes(
				attribute.Bool("ovm.discover.hang", true),
			)
			span.End()
		}
	}()

	// Set up handling for the items and errors that are returned before they
	// are passed back to the caller
	var numItems atomic.Int32
	var numErrs atomic.Int32
	var itemHandler ItemHandler = func(item *sdp.Item) {
		if item == nil {
			return
		}

		if err := item.Validate(); err != nil {
			span.RecordError(err)
			responses <- sdp.NewQueryResponseFromError(&sdp.QueryError{
				UUID:          q.GetUUID(),
				ErrorType:     sdp.QueryError_OTHER,
				ErrorString:   err.Error(),
				Scope:         q.GetScope(),
				ResponderName: e.EngineConfig.SourceName,
				ItemType:      q.GetType(),
			})
			return
		}

		// Store metadata
		item.Metadata = &sdp.Metadata{
			Timestamp:   timestamppb.New(time.Now()),
			SourceName:  adapter.Name(),
			SourceQuery: q,
		}

		// Mark the item as hidden if the adapter is hidden
		if hs, ok := adapter.(HiddenAdapter); ok {
			item.Metadata.Hidden = hs.Hidden()
		}

		// Send the item back to the caller
		numItems.Add(1)
		responses <- sdp.NewQueryResponseFromItem(item)
	}
	var errHandler ErrHandler = func(err error) {
		if err == nil {
			return
		}
		// add a recover to prevent panic from stream error handler.
		defer tracing.LogRecoverToReturn(ctx, "StreamErrorHandler")

		// Record the error in the trace
		span.RecordError(err, trace.WithStackTrace(true))

		// Send the error back to the caller
		numErrs.Add(1)
		responses <- queryResponseFromError(err, q, adapter, e.EngineConfig.SourceName)
	}
	stream := NewQueryResultStream(itemHandler, errHandler)

	// Check that our context is okay before doing anything expensive
	if ctx.Err() != nil {
		span.RecordError(ctx.Err())

		responses <- sdp.NewQueryResponseFromError(&sdp.QueryError{
			UUID:          q.GetUUID(),
			ErrorType:     sdp.QueryError_OTHER,
			ErrorString:   ctx.Err().Error(),
			Scope:         q.GetScope(),
			ResponderName: e.EngineConfig.SourceName,
			ItemType:      q.GetType(),
		})
		return
	}

	switch q.GetMethod() {
	case sdp.QueryMethod_GET:
		newItem, err := adapter.Get(ctx, q.GetScope(), q.GetQuery(), q.GetIgnoreCache())

		if newItem != nil {
			stream.SendItem(newItem)
		}
		if err != nil {
			stream.SendError(err)
		}
	case sdp.QueryMethod_LIST:
		if listStreamingAdapter, ok := adapter.(ListStreamableAdapter); ok {
			// Prefer the streaming methods if they are available
			listStreamingAdapter.ListStream(ctx, q.GetScope(), q.GetIgnoreCache(), stream)
		} else if listableAdapter, ok := adapter.(ListableAdapter); ok {
			// Fall back to the non-streaming methods
			resultItems, err := listableAdapter.List(ctx, q.GetScope(), q.GetIgnoreCache())

			for _, i := range resultItems {
				stream.SendItem(i)
			}
			if err != nil {
				stream.SendError(err)
			}
		} else {
			// Log the error instead of sending it over the stream
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.adapter.name":  adapter.Name(),
				"ovm.adapter.type":  q.GetType(),
				"ovm.adapter.scope": q.GetScope(),
			}).Warn("adapter is not listable")
		}
	case sdp.QueryMethod_SEARCH:
		if searchStreamingAdapter, ok := adapter.(SearchStreamableAdapter); ok {
			// Prefer the streaming methods if they are available
			searchStreamingAdapter.SearchStream(ctx, q.GetScope(), q.GetQuery(), q.GetIgnoreCache(), stream)
		} else if searchableAdapter, ok := adapter.(SearchableAdapter); ok {
			// Fall back to the non-streaming methods
			resultItems, err := searchableAdapter.Search(ctx, q.GetScope(), q.GetQuery(), q.GetIgnoreCache())

			for _, i := range resultItems {
				stream.SendItem(i)
			}
			if err != nil {
				stream.SendError(err)
			}
		} else {
			// Log the error instead of sending it over the stream
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.adapter.name":  adapter.Name(),
				"ovm.adapter.type":  q.GetType(),
				"ovm.adapter.scope": q.GetScope(),
			}).Warn("adapter is not searchable")
		}
	}

	span.SetAttributes(
		attribute.Int("ovm.adapter.numItems", int(numItems.Load())),
		attribute.Int("ovm.adapter.numErrors", int(numErrs.Load())),
	)
}

// queryResponseFromError converts an error into a QueryResponse. This takes
// care to not double-wrap `sdp.QueryError` errors.
func queryResponseFromError(err error, q *sdp.Query, adapter Adapter, sourceName string) *sdp.QueryResponse {
	var sdpErr *sdp.QueryError
	if !errors.As(err, &sdpErr) {
		sdpErr = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	// Add details that might not be populated by the adapter
	sdpErr.Scope = q.GetScope()
	sdpErr.UUID = q.GetUUID()
	sdpErr.SourceName = adapter.Name()
	sdpErr.ItemType = adapter.Metadata().GetType()
	sdpErr.ResponderName = sourceName

	return sdp.NewQueryResponseFromError(sdpErr)
}
