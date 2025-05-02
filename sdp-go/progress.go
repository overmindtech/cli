package sdp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DefaultResponseInterval is the default period of time within which responses
// are sent (5 seconds)
const DefaultResponseInterval = (5 * time.Second)

// DefaultStartTimeout is the default period of time to wait for the first
// response on a query. If no response is received in this time, the query will
// be marked as complete.
const DefaultStartTimeout = 500 * time.Millisecond

// ResponseSender is a struct responsible for sending responses out on behalf of
// agents that are working on that request. Think of it as the agent side
// component of Responder
type ResponseSender struct {
	// How often to send responses. The expected next update will be 230% of
	// this value, allowing for one-and-a-bit missed responses before it is
	// marked as stalled
	ResponseInterval time.Duration
	ResponseSubject  string
	monitorRunning   sync.WaitGroup
	monitorKill      chan *Response // Sending to this channel will kill the response sender goroutine and publish the sent message as last msg on the subject
	responderName    string
	responderId      uuid.UUID
	connection       EncodedConnection
	responseCtx      context.Context
}

// Start sends the first response on the given subject and connection to say
// that the request is being worked on. It also starts a go routine to continue
// sending responses.
//
// The user should make sure to call Done(), Error() or Cancel() once the query
// has finished to make sure this process stops sending responses. The sender
// will also be stopped if the context is cancelled
func (rs *ResponseSender) Start(ctx context.Context, ec EncodedConnection, responderName string, responderId uuid.UUID) {
	rs.monitorKill = make(chan *Response, 1)
	rs.responseCtx = ctx

	// Set the default if it's not set
	if rs.ResponseInterval == 0 {
		rs.ResponseInterval = DefaultResponseInterval
	}

	// Tell it to expect the next update in 230% of the expected time. This
	// allows for a response getting lost, plus some delay
	nextUpdateIn := durationpb.New(time.Duration((float64(rs.ResponseInterval) * 2.3)))

	// Set struct values
	rs.responderName = responderName
	rs.responderId = responderId
	rs.connection = ec

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Responder:     rs.responderName,
		ResponderUUID: rs.responderId[:],
		State:         ResponderState_WORKING,
		NextUpdateIn:  nextUpdateIn,
	}

	if rs.connection != nil {
		// Send the initial response
		err := rs.connection.Publish(
			ctx,
			rs.ResponseSubject,
			&QueryResponse{ResponseType: &QueryResponse_Response{Response: &resp}},
		)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("Error publishing initial response")
		}
	}

	rs.monitorRunning.Add(1)

	// Start a goroutine to send further responses
	go func() {
		defer tracing.LogRecoverToReturn(ctx, "ResponseSender ticker")
		// confirm closure on exit
		defer rs.monitorRunning.Done()

		if ec == nil {
			return
		}
		tick := time.NewTicker(rs.ResponseInterval)
		defer tick.Stop()

		for {
			var err error

			select {
			case <-rs.monitorKill:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				err = rs.connection.Publish(
					ctx,
					rs.ResponseSubject,
					&QueryResponse{ResponseType: &QueryResponse_Response{Response: &resp}},
				)

				if err != nil {
					log.WithContext(ctx).WithError(err).Error("Error publishing response")
				}
			}
		}
	}()
}

// Kill Kills the response sender immediately. This should be used if something
// has failed and you don't want to send a completed response
//
// Deprecated: Use KillWithContext(ctx) instead
func (rs *ResponseSender) Kill() {
	rs.killWithResponse(context.Background(), nil)
}

// KillWithContext Kills the response sender immediately. This should be used if something
// has failed and you don't want to send a completed response
func (rs *ResponseSender) KillWithContext(ctx context.Context) {
	rs.killWithResponse(ctx, nil)
}

func (rs *ResponseSender) killWithResponse(ctx context.Context, r *Response) {
	// close the channel to kill the sender
	close(rs.monitorKill)

	// wait for the sender to be actually done
	rs.monitorRunning.Wait()

	if rs.connection != nil {
		if r != nil {
			// Send the final response
			err := rs.connection.Publish(ctx, rs.ResponseSubject, &QueryResponse{
				ResponseType: &QueryResponse_Response{
					Response: r,
				},
			})
			if err != nil {
				log.WithContext(ctx).WithError(err).Error("Error publishing final response")
			}
		}
	}
}

// Done kills the responder but sends a final completion message
//
// Deprecated: Use DoneWithContext(ctx) instead
func (rs *ResponseSender) Done() {
	rs.DoneWithContext(context.Background())
}

// DoneWithContext kills the responder but sends a final completion message
func (rs *ResponseSender) DoneWithContext(ctx context.Context) {
	resp := Response{
		Responder:     rs.responderName,
		ResponderUUID: rs.responderId[:],
		State:         ResponderState_COMPLETE,
	}
	rs.killWithResponse(ctx, &resp)
}

// Error marks the request and completed with error, and sends the final error
// response
//
// Deprecated: Use ErrorWithContext(ctx) instead
func (rs *ResponseSender) Error() {
	rs.ErrorWithContext(context.Background())
}

// ErrorWithContext marks the request and completed with error, and sends the final error
// response
func (rs *ResponseSender) ErrorWithContext(ctx context.Context) {
	resp := Response{
		Responder:     rs.responderName,
		ResponderUUID: rs.responderId[:],
		State:         ResponderState_ERROR,
	}
	rs.killWithResponse(ctx, &resp)
}

// Cancel Marks the request as CANCELLED and sends the final response
//
// Deprecated: Use CancelWithContext(ctx) instead
func (rs *ResponseSender) Cancel() {
	rs.CancelWithContext(context.Background())
}

// CancelWithContext Marks the request as CANCELLED and sends the final response
func (rs *ResponseSender) CancelWithContext(ctx context.Context) {
	resp := Response{
		Responder:     rs.responderName,
		ResponderUUID: rs.responderId[:],
		State:         ResponderState_CANCELLED,
	}
	rs.killWithResponse(ctx, &resp)
}

type lastResponse struct {
	Response  *Response
	Timestamp time.Time
}

// Checks to see if this responder is stalled. If it is, it will update the
// responder state to ResponderState_STALLED. Only runs if the responder is in
// the WORKING state, doesn't do anything otherwise.
func (l *lastResponse) checkStalled() {
	if l.Response == nil || l.Response.GetState() != ResponderState_WORKING {
		return
	}

	// Calculate if it's stalled, but only if it has a `NextUpdateIn` value.
	// Responders that do not provided a `NextUpdateIn` value are not considered
	// for stalling
	timeSinceLastUpdate := time.Since(l.Timestamp)
	timeToNextUpdate := l.Response.GetNextUpdateIn().AsDuration()
	if timeToNextUpdate > 0 && timeSinceLastUpdate > timeToNextUpdate {
		l.Response.State = ResponderState_STALLED
	}
}

// SourceQuery represents the status of a query
type SourceQuery struct {
	// A map of ResponderUUIDs to the last response we got from them
	responders   map[uuid.UUID]*lastResponse
	respondersMu sync.Mutex

	// Channel storage for sending back to the user
	responseChan chan<- *QueryResponse

	// Use to make sure a user doesn't try to start a request twice. This is an
	// atomic to allow tests to directly inject messages using
	// `handleQueryResponse`
	startTimeoutElapsed atomic.Bool

	querySub *nats.Subscription

	cancel context.CancelFunc
}

// The current progress of the tracked query
type SourceQueryProgress struct {
	// How many responders are currently working on this query. This means they
	// are active sending updates
	Working int

	// Stalled responders are ones that have sent updates in the past, but the
	// latest update is overdue. This likely indicates a problem with the
	// responder
	Stalled int

	// Responders that are complete
	Complete int

	// Responders that failed
	Error int

	// Responders that were cancelled. When cancelling the SourceQueryProgress
	// does not wait for responders to acknowledge the cancellation, it simply
	// sends the message and marks all responders that are currently "working"
	// as "cancelled". It is possible that a responder will self-report
	// cancellation, but given the timings this is unlikely as it would need to
	// be very fast
	Cancelled int

	// The total number of tracked responders
	Responders int
}

// RunSourceQuery returns a pointer to a SourceQuery object with the various
// internal members initialized. A startTimeout must also be provided, feel free
// to use `DefaultStartTimeout` if you don't have a specific value in mind.
func RunSourceQuery(ctx context.Context, query *Query, startTimeout time.Duration, ec EncodedConnection, responseChan chan<- *QueryResponse) (*SourceQuery, error) {
	if startTimeout == 0 {
		return nil, errors.New("startTimeout must be greater than 0")
	}

	if ec.Underlying() == nil {
		return nil, errors.New("nil NATS connection")
	}

	if responseChan == nil {
		return nil, errors.New("nil response channel")
	}

	if query.GetScope() == "" {
		return nil, errors.New("cannot execute request with blank scope")
	}

	// Generate a UUID if required
	if len(query.GetUUID()) == 0 {
		u := uuid.New()
		query.UUID = u[:]
	}

	// Calculate the correct subject to send the message on
	var requestSubject string
	if query.GetScope() == WILDCARD {
		requestSubject = "request.all"
	} else {
		requestSubject = fmt.Sprintf("request.scope.%v", query.GetScope())
	}

	// Create the channel that NATS responses will come through
	natsResponses := make(chan *QueryResponse)

	// Create a timer for the start timeout
	startTimeoutTimer := time.NewTimer(startTimeout)

	// Subscribe to the query subject and wait for responses
	querySub, err := ec.Subscribe(query.Subject(), NewQueryResponseHandler("", func(ctx context.Context, qr *QueryResponse) { //nolint:contextcheck // we pass the context in the func
		natsResponses <- qr
	}))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	sq := &SourceQuery{
		responders:          make(map[uuid.UUID]*lastResponse),
		startTimeoutElapsed: atomic.Bool{},
		querySub:            querySub,
		cancel:              cancel,
		responseChan:        responseChan,
	}

	// Main processing loop. This runs is the main goroutine that tracks this
	// request
	go func() {
		// Initialise the stall check ticker
		stallCheck := time.NewTicker(500 * time.Millisecond)
		defer stallCheck.Stop()
		ctx, span := tracing.Tracer().Start(ctx, "QueryProgress")
		defer span.End()

		// Attach query information to the span
		span.SetAttributes(
			attribute.String("ovm.sdp.type", query.GetType()),
			attribute.String("ovm.sdp.scope", query.GetScope()),
			attribute.String("ovm.sdp.uuid", uuid.UUID(query.GetUUID()).String()),
			attribute.String("ovm.sdp.method", query.GetMethod().String()),
		)

		for {
			select {
			case <-ctx.Done():
				// Since this context is done, we need a new context just to
				// send the cancellation message
				cancelCtx, cancelCtxCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
				defer cancelCtxCancel()

				// Send a cancel message to all responders
				cancelRequest := CancelQuery{
					UUID: query.GetUUID(),
				}

				var cancelSubject string
				if query.GetScope() == WILDCARD {
					cancelSubject = "cancel.all"
				} else {
					cancelSubject = fmt.Sprintf("cancel.scope.%v", query.GetScope())
				}

				err := ec.Publish(cancelCtx, cancelSubject, &cancelRequest)

				if err != nil {
					log.WithContext(ctx).WithError(err).Error("Error sending cancel message")
					span.RecordError(err)
				}

				sq.markWorkingRespondersCancelled()
				sq.cleanup(ctx)
				return
			case <-startTimeoutTimer.C:
				sq.startTimeoutElapsed.Store(true)

				if sq.finished() {
					sq.cleanup(ctx)
					return
				}
			case response := <-natsResponses:
				// Handle the response
				if sq.handleQueryResponse(ctx, response) {
					// This means we are done
					return
				}
			case <-stallCheck.C:

				// If we get here, it means that we haven't had a response
				// in a while, so we should check to see if things have
				// stalled
				if sq.finished() {
					sq.cleanup(ctx)
					return
				}
			}
		}
	}()

	// Send the message to start the query
	err = ec.Publish(ctx, requestSubject, query)
	if err != nil {
		return nil, err
	}

	return sq, nil
}

// Execute a given request and wait for it to finish, returns the items that
// were found and any errors. The third return error value will only be returned
// only if there is a problem making the request. Details of which responders
// have failed etc. should be determined using the typical methods like
// `NumError()`.
func RunSourceQuerySync(ctx context.Context, query *Query, startTimeout time.Duration, ec EncodedConnection) ([]*Item, []*Edge, []*QueryError, error) {
	items := make([]*Item, 0)
	edges := make([]*Edge, 0)
	errs := make([]*QueryError, 0)
	r := make(chan *QueryResponse, 128)

	if ec == nil {
		return items, edges, errs, errors.New("nil NATS connection")
	}

	_, err := RunSourceQuery(ctx, query, startTimeout, ec, r)
	if err != nil {
		return items, edges, errs, err
	}

	// Read items and errors
	for response := range r {
		item := response.GetNewItem()
		if item != nil {
			items = append(items, item)
		}
		edge := response.GetEdge()
		if edge != nil {
			edges = append(edges, edge)
		}
		qErr := response.GetError()
		if qErr != nil {
			errs = append(errs, qErr)
		}
		// ignore status responses for now
		// status := response.GetResponse()
		// if status != nil {
		// 	panic("qp: status not implemented yet")
		// }
	}

	// when the channel closes, we're done
	return items, edges, errs, nil
}

// Cancels the request, sending a cancel message to all responders and closing
// the response channel. The query can also be cancelled by cancelling the
// context that was passed in the `Start` method
func (sq *SourceQuery) Cancel() {
	sq.cancel()
}

// This is split out into its own function so that it can be tested more easily
// with out having to worry about race conditions. This returns a boolean which
// indicates if the request is complete or not
func (sq *SourceQuery) handleQueryResponse(ctx context.Context, response *QueryResponse) bool {
	switch r := response.GetResponseType().(type) {
	case *QueryResponse_NewItem:
		sq.handleItem(r.NewItem)
	case *QueryResponse_Edge:
		sq.handleEdge(r.Edge)
	case *QueryResponse_Error:
		sq.handleError(r.Error)
	case *QueryResponse_Response:
		sq.handleResponse(ctx, r.Response)

		if sq.finished() {
			sq.cleanup(ctx)
			return true
		}
	}

	return false
}

// markWorkingRespondersCancelled marks all working responders as cancelled
// internally, there is no need to wait for them to confirm the cancellation, as
// we're not going to wait for any further responses.
func (sq *SourceQuery) markWorkingRespondersCancelled() {
	sq.respondersMu.Lock()
	defer sq.respondersMu.Unlock()

	for _, lastResponse := range sq.responders {
		if lastResponse.Response.GetState() == ResponderState_WORKING {
			lastResponse.Response.State = ResponderState_CANCELLED
		}
	}
}

// Whether the query should be considered finished or not. This is based on
// whether the start timeout has elapsed and all responders are done
func (sq *SourceQuery) finished() bool {
	return sq.startTimeoutElapsed.Load() && sq.allDone()
}

// Cleans up the query, unsubscribing from the query subject and closing the
// response channel
func (sq *SourceQuery) cleanup(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if sq.querySub != nil {
		err := sq.querySub.Unsubscribe()
		if err != nil {
			log.WithField("error", err).Error("Error unsubscribing from query subject")
			span.RecordError(err)
		}
	}

	close(sq.responseChan)
	sq.cancel()
}

// Sends the item back to the response channel, also extracts and synthesises
// edges from `LinkedItems` and `LinkedItemQueries` and sends them back too
func (sq *SourceQuery) handleItem(item *Item) {
	if item == nil {
		return
	}

	// Send the item back over the channel
	// TODO(LIQs): translation is not necessary anymore; update code and method comment
	item, edges := TranslateLinksToEdges(item)
	sq.responseChan <- &QueryResponse{
		ResponseType: &QueryResponse_NewItem{NewItem: item},
	}
	for _, e := range edges {
		sq.responseChan <- &QueryResponse{
			ResponseType: &QueryResponse_Edge{Edge: e},
		}
	}
}

// Sends the edge back to the response channel
func (sq *SourceQuery) handleEdge(edge *Edge) {
	if edge == nil {
		return
	}

	sq.responseChan <- &QueryResponse{
		ResponseType: &QueryResponse_Edge{Edge: edge},
	}
}

// Send the error back to the response channel
func (sq *SourceQuery) handleError(err *QueryError) {
	if err == nil {
		return
	}

	sq.responseChan <- &QueryResponse{
		ResponseType: &QueryResponse_Error{
			Error: err,
		},
	}
}

// Update the internal state with the response
func (sq *SourceQuery) handleResponse(ctx context.Context, response *Response) {
	span := trace.SpanFromContext(ctx)

	// do not deal with responses that do not have a responder UUID
	ru, err := uuid.FromBytes(response.GetResponderUUID())
	if err != nil {
		span.RecordError(fmt.Errorf("error parsing responder UUID: %w", err))
		return
	}

	sq.respondersMu.Lock()
	defer sq.respondersMu.Unlock()

	// Protect against out-of order responses. Do not mark a responder as
	// working if it has already finished. this should never happen, but we want
	// to know if it does as it will indicate a bug in the responder itself
	last, exists := sq.responders[ru]
	if exists {
		if last.Response != nil {
			switch last.Response.GetState() {
			case ResponderState_COMPLETE, ResponderState_ERROR, ResponderState_CANCELLED:
				err = fmt.Errorf("out-of-order response. Responder was already in the state %v, skipping update to %v", last.Response.String(), response.GetState().String())
				span.RecordError(err)
				sentry.CaptureException(err)
				return
			case ResponderState_WORKING, ResponderState_STALLED:
				// This is fine, we can update the state
			}
		}
	}

	// Update the stored data
	sq.responders[ru] = &lastResponse{
		Response:  response,
		Timestamp: time.Now(),
	}
}

// Checks whether all responders are done or not. A "Done" responder is one that
// is either: Complete, Error, Cancelled or Stalled
//
// Note that this doesn't perform locking if the mutex, this needs to be done by
// the caller
func (sq *SourceQuery) allDone() bool {
	sq.respondersMu.Lock()
	defer sq.respondersMu.Unlock()

	for _, lastResponse := range sq.responders {
		// Recalculate the stall status
		lastResponse.checkStalled()

		if lastResponse.Response.GetState() == ResponderState_WORKING {
			return false
		}
	}

	return true
}

// TranslateLinksToEdges Translates linked items and queries into edges. This is
// a temporary stop gap measure to allow parallel processing of items and edges
// in the gateway while allowing other parts of the system to be updated
// independently. See https://github.com/overmindtech/workspace/issues/753
func TranslateLinksToEdges(item *Item) (*Item, []*Edge) {
	// TODO(LIQs): translation is not necessary anymore; delete this method and all callsites
	lis := item.GetLinkedItems()
	item.LinkedItems = nil
	liqs := item.GetLinkedItemQueries()
	item.LinkedItemQueries = nil

	edges := []*Edge{}

	for _, li := range lis {
		edges = append(edges, &Edge{
			From:             item.Reference(),
			To:               li.GetItem(),
			BlastPropagation: li.GetBlastPropagation(),
		})
	}

	for _, liq := range liqs {
		edges = append(edges, &Edge{
			From:             item.Reference(),
			To:               liq.GetQuery().Reference(),
			BlastPropagation: liq.GetBlastPropagation(),
		})
	}

	return item, edges
}

func (sq *SourceQuery) Progress() SourceQueryProgress {
	sq.respondersMu.Lock()
	defer sq.respondersMu.Unlock()

	var numWorking, numStalled, numComplete, numError, numCancelled int

	// Loop over all responders once and calculate the progress
	for _, lastResponse := range sq.responders {
		// Recalculate the stall status
		lastResponse.checkStalled()

		switch lastResponse.Response.GetState() {
		case ResponderState_WORKING:
			numWorking++
		case ResponderState_STALLED:
			numStalled++
		case ResponderState_COMPLETE:
			numComplete++
		case ResponderState_ERROR:
			numError++
		case ResponderState_CANCELLED:
			numCancelled++
		}
	}

	return SourceQueryProgress{
		Working:    numWorking,
		Stalled:    numStalled,
		Complete:   numComplete,
		Error:      numError,
		Cancelled:  numCancelled,
		Responders: len(sq.responders),
	}
}

func (sq *SourceQuery) String() string {
	progress := sq.Progress()

	return fmt.Sprintf(
		"Working: %v\nStalled: %v\nComplete: %v\nError: %v\nCancelled: %v\nResponders: %v\n",
		progress.Working,
		progress.Stalled,
		progress.Complete,
		progress.Error,
		progress.Cancelled,
		progress.Responders,
	)
}
