package sdp

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DefaultResponseInterval is the default period of time within which responses
// are sent (5 seconds)
const DefaultResponseInterval = (5 * time.Second)

// DefaultStartTimeout is the default period of time to wait for the first
// response on a query. If no response is received in this time, the query will
// be marked as complete.
const DefaultStartTimeout = 500 * time.Millisecond

// DefaultDrainDelay How long to wait after all is complete before draining all
// NATS connections
const DefaultDrainDelay = (100 * time.Millisecond)

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

// responderStatus represents the status of a responder
type responderStatus struct {
	Name           string
	ID             uuid.UUID
	monitorContext context.Context
	monitorCancel  context.CancelFunc
	lastState      ResponderState
	lastStateTime  time.Time
	mutex          sync.RWMutex
}

// CancelMonitor Cancels the running stall monitor goroutine if there is one
func (re *responderStatus) CancelMonitor() {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	if re.monitorCancel != nil {
		re.monitorCancel()
	}
}

// SetMonitorContext Saves the context details for the monitor goroutine so that
// it can be cancelled later, freeing up resources
func (re *responderStatus) SetMonitorContext(ctx context.Context, cancel context.CancelFunc) {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	re.monitorContext = ctx
	re.monitorCancel = cancel
}

// SetState updates the state and last state time of the responder
func (re *responderStatus) SetState(s ResponderState) {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	re.lastState = s
	re.lastStateTime = time.Now()
}

// LastState Returns the last state response for a given responder
func (re *responderStatus) LastState() ResponderState {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	return re.lastState
}

// LastStateTime Returns the last state response for a given responder
func (re *responderStatus) LastStateTime() time.Time {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	return re.lastStateTime
}

// QueryProgress represents the status of a query
type QueryProgress struct {
	// How long to wait after `MarkStarted()` has been called to get at least
	// one responder, if there are no responders in this time, the request will
	// be marked as completed
	StartTimeout        time.Duration
	StartTimeoutElapsed atomic.Bool // Whether the start timeout has elapsed
	Query               *Query
	requestCtx          context.Context

	// How long to wait before draining NATS connections after all have
	// completed
	DrainDelay time.Duration

	responders      map[uuid.UUID]*responderStatus
	respondersMutex sync.RWMutex

	// Channel storage for sending back to the user
	responseChan   chan<- *QueryResponse
	doneChan       chan struct{} // Closed when request is fully complete
	chanMutex      sync.RWMutex
	channelsClosed bool // Additional protection against send on closed chan. This isn't brilliant but I can't think of a better way at the moment
	drain          sync.Once
	drainStack     []byte

	started   bool
	cancelled bool
	subMutex  sync.Mutex

	querySub *nats.Subscription

	// Counters for how many things we have sent over the channels. This is
	// required to make sure that we aren't closing channels that have pending
	// things to be sent on them
	itemsProcessed  *int64
	errorsProcessed *int64

	noResponderContext context.Context
	noRespondersCancel context.CancelFunc
}

// NewQueryProgress returns a pointer to a QueryProgress object with the various
// internal members initialized. A startTimeout must also be provided, however
// if it's nil it will default to `DefaultStartTimeout`
func NewQueryProgress(q *Query, startTimeout time.Duration) *QueryProgress {

	var finalTimeout time.Duration

	// Ensure that we time out eventually if nothing responds
	if startTimeout == 0 {
		finalTimeout = DefaultStartTimeout
	} else {
		finalTimeout = startTimeout
	}

	return &QueryProgress{
		Query:           q,
		StartTimeout:    finalTimeout,
		DrainDelay:      DefaultDrainDelay,
		responders:      make(map[uuid.UUID]*responderStatus),
		doneChan:        make(chan struct{}),
		itemsProcessed:  new(int64),
		errorsProcessed: new(int64),
	}
}

// Start starts a given request, sending items to the supplied itemChannel. It
// is up to the user to watch for completion. When the request does complete,
// the NATS subscriptions will automatically drain and the itemChannel will be
// closed.
//
// The fact that the items chan is closed when all items have been received
// means that the only thing a user needs to do in order to process all items
// and then continue is range over the channel e.g.
//
//	for item := range itemChannel {
//		// Do something with the item
//		fmt.Println(item)
//
//		// This loop  will exit once the request is finished
//	}
func (qp *QueryProgress) Start(ctx context.Context, ec EncodedConnection, responseChan chan<- *QueryResponse) error {
	if qp.started {
		return errors.New("already started")
	}

	if ec.Underlying() == nil {
		return errors.New("nil NATS connection")
	}

	if responseChan == nil {
		return errors.New("nil response channel")
	}

	qp.requestCtx = ctx

	if len(qp.Query.GetUUID()) == 0 {
		u := uuid.New()
		qp.Query.UUID = u[:]
	}

	var requestSubject string

	if qp.Query.GetScope() == "" {
		return errors.New("cannot execute request with blank scope")
	}

	if qp.Query.GetScope() == WILDCARD {
		requestSubject = "request.all"
	} else {
		requestSubject = fmt.Sprintf("request.scope.%v", qp.Query.GetScope())
	}

	// Store the channels
	qp.chanMutex.Lock()
	defer qp.chanMutex.Unlock()
	qp.responseChan = responseChan

	qp.subMutex.Lock()
	defer qp.subMutex.Unlock()

	var err error

	itemHandler := func(ctx context.Context, item *Item) {
		defer atomic.AddInt64(qp.itemsProcessed, 1)

		span := trace.SpanFromContext(ctx)

		if item == nil {
			span.SetAttributes(
				attribute.String("ovm.item", "nil"),
			)
		} else {
			span.SetAttributes(
				attribute.String("ovm.item", item.GloballyUniqueName()),
			)

			qp.chanMutex.RLock()
			defer qp.chanMutex.RUnlock()
			if qp.channelsClosed {
				var itemTime time.Time

				if item.GetMetadata() != nil {
					itemTime = item.GetMetadata().GetTimestamp().AsTime()
				}

				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				log.WithContext(ctx).WithFields(log.Fields{
					"Type":                 item.GetType(),
					"Scope":                item.GetScope(),
					"UniqueAttributeValue": item.UniqueAttributeValue(),
					"Item Timestamp":       itemTime.String(),
					"Current Time":         time.Now().String(),
					"Stack":                string(qp.drainStack),
				}).Error("SDP-GO ERROR: An Item was processed after Drain() was called. Please add these details to: https://github.com/overmindtech/cli/sdp-go/issues/15.")

				span.SetStatus(codes.Error, "SDP-GO ERROR: An Item was processed after Drain() was called. Please add these details to: https://github.com/overmindtech/cli/sdp-go/issues/15.")
				return
			}

			// TODO: extract linked items and linked item queries and pass them on
			qp.responseChan <- &QueryResponse{
				ResponseType: &QueryResponse_NewItem{NewItem: item},
			}

		}
	}

	errorHandler := func(ctx context.Context, qErr *QueryError) {
		defer atomic.AddInt64(qp.errorsProcessed, 1)

		if qErr != nil {
			span := trace.SpanFromContext(ctx)
			span.SetStatus(codes.Error, qErr.Error())
			span.SetAttributes(
				attribute.Int64("ovm.sdp.errorsProcessed", *qp.errorsProcessed),
				attribute.String("ovm.sdp.errorString", qErr.GetErrorString()),
				attribute.String("ovm.sdp.errorType", qErr.GetErrorType().String()),
				attribute.String("ovm.scope", qErr.GetScope()),
				attribute.String("ovm.type", qErr.GetItemType()),
				attribute.String("ovm.sdp.sourceName", qErr.GetSourceName()),
				attribute.String("ovm.sdp.responderName", qErr.GetResponderName()),
			)

			qp.chanMutex.RLock()
			defer qp.chanMutex.RUnlock()
			if qp.channelsClosed {
				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				log.WithContext(ctx).WithFields(log.Fields{
					"UUID":          qErr.GetUUID(),
					"ErrorType":     qErr.GetErrorType(),
					"ErrorString":   qErr.GetErrorString(),
					"Scope":         qErr.GetScope(),
					"SourceName":    qErr.GetSourceName(),
					"ItemType":      qErr.GetItemType(),
					"ResponderName": qErr.GetResponderName(),
				}).Error("SDP-GO ERROR: A QueryError was processed after Drain() was called. Please add these details to: https://github.com/overmindtech/cli/sdp-go/issues/15.")
				return
			}

			qp.responseChan <- &QueryResponse{
				ResponseType: &QueryResponse_Error{
					Error: qErr,
				},
			}
		}
	}

	qp.querySub, err = ec.Subscribe(qp.Query.Subject(), NewQueryResponseHandler("", func(ctx context.Context, qr *QueryResponse) { //nolint:contextcheck // we pass the context in the func
		log.WithContext(ctx).WithFields(log.Fields{
			"response": qr,
		}).Trace("Received response")
		switch qr.GetResponseType().(type) {
		case *QueryResponse_NewItem:
			itemHandler(ctx, qr.GetNewItem())
		case *QueryResponse_Error:
			errorHandler(ctx, qr.GetError())
		case *QueryResponse_Response:
			qp.ProcessResponse(ctx, qr.GetResponse())
		default:
			panic(fmt.Sprintf("Received unexpected QueryResponse: %v", qr))
		}
	}))
	if err != nil {
		return err
	}

	err = ec.Publish(ctx, requestSubject, qp.Query)

	qp.markStarted()

	if err != nil {
		return err
	}

	return nil
}

// markStarted Marks the request as started and will cause it to be marked as
// done if there are no responders after StartTimeout duration
func (qp *QueryProgress) markStarted() {
	// We're using this mutex to also lock access to the context and cancel
	qp.respondersMutex.Lock()
	defer qp.respondersMutex.Unlock()

	qp.started = true
	qp.noResponderContext, qp.noRespondersCancel = context.WithCancel(context.Background())

	var startTimeout *time.Timer
	if qp.StartTimeout == 0 {
		startTimeout = time.NewTimer(1 * time.Second)
	} else {
		startTimeout = time.NewTimer(qp.StartTimeout)
	}

	go func(ctx context.Context) {
		defer tracing.LogRecoverToReturn(ctx, "QueryProgress startTimeout")
		select {
		case <-startTimeout.C:
			qp.StartTimeoutElapsed.Store(true)

			// Once the start timeout has elapsed, if there are no
			// responders, or all of them are done, we can drain the
			// connections and mark everything as done
			qp.respondersMutex.RLock()
			defer qp.respondersMutex.RUnlock()

			if qp.numResponders() == 0 || qp.allDone() {
				qp.Drain()
			}
		case <-ctx.Done():
			startTimeout.Stop()
		}
	}(qp.noResponderContext)
}

// Drain Tries to drain connections gracefully. If not though, connections are
// forcibly closed and the item and error channels closed
func (qp *QueryProgress) Drain() {
	// Use sync.Once to ensure that if this is called in parallel goroutines it
	// isn't run twice
	qp.drain.Do(func() {
		qp.subMutex.Lock()
		defer qp.subMutex.Unlock()

		qp.drainStack = debug.Stack()

		if qp.noRespondersCancel != nil {
			// Cancel the no responders watcher to release the resources
			qp.noRespondersCancel()
		}

		// Close the item and error subscriptions
		err := unsubscribeGracefully(qp.querySub)
		if err != nil {
			log.WithContext(qp.requestCtx).WithError(err).Error("Error unsubscribing from query subject")
		}

		qp.chanMutex.Lock()
		defer qp.chanMutex.Unlock()

		if qp.responseChan != nil {
			close(qp.responseChan)
		}

		// Only if the drain is fully complete should we close the doneChan
		close(qp.doneChan)

		qp.channelsClosed = true
	})
}

// Done Returns a channel when the request is fully complete and all channels
// closed
func (qp *QueryProgress) Done() <-chan struct{} {
	return qp.doneChan
}

// Cancel Cancels a request and waits for all responders to report that they
// were finished, cancelled or to be marked as stalled. If the context expires
// before this happens, the request is cancelled forcibly, with subscriptions
// being removed and channels closed. This method will only return when
// cancellation is complete
//
// Returns a boolean indicating whether the cancellation needed to be forced
func (qp *QueryProgress) Cancel(ctx context.Context, ec EncodedConnection) bool {
	err := qp.AsyncCancel(ec)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Error cancelling request")
	}

	select {
	case <-qp.Done():
		// If the request finishes gracefully, that's good
		return false
	case <-ctx.Done():
		// If the context is cancelled first, then force the draining
		qp.Drain()
		return true
	}
}

// Cancel Sends a cancellation request for a given request
func (qp *QueryProgress) AsyncCancel(ec EncodedConnection) error {
	if ec == nil {
		return errors.New("nil NATS connection")
	}

	cancelRequest := CancelQuery{
		UUID: qp.Query.GetUUID(),
	}

	var cancelSubject string

	if qp.Query.GetScope() == WILDCARD {
		cancelSubject = "cancel.all"
	} else {
		cancelSubject = fmt.Sprintf("cancel.scope.%v", qp.Query.GetScope())
	}

	qp.cancelled = true

	err := ec.Publish(qp.requestCtx, cancelSubject, &cancelRequest)

	if err != nil {
		return err
	}

	// Check this immediately in case nothing had started yet
	if qp.allDone() {
		qp.Drain()
	}

	return nil
}

// Execute Executes a given request and waits for it to finish, returns the
// items that were found and any errors. The third return error value  will only
// be returned only if there is a problem making the request. Details of which
// responders have failed etc. should be determined using the typical methods
// like `NumError()`.
func (qp *QueryProgress) Execute(ctx context.Context, ec EncodedConnection) ([]*Item, []*QueryError, error) {
	items := make([]*Item, 0)
	errs := make([]*QueryError, 0)
	r := make(chan *QueryResponse)

	if ec == nil {
		return items, errs, errors.New("nil NATS connection")
	}

	err := qp.Start(ctx, ec, r)

	if err != nil {
		return items, errs, err
	}

	for {
		// Read items and errors
		select {
		case response, ok := <-r:
			if !ok {
				// when the channel closes, we're done
				return items, errs, nil
			}
			item := response.GetNewItem()
			if item != nil {
				items = append(items, item)
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
	}
}

// ProcessResponse processes an SDP Response and updates the database
// accordingly
func (qp *QueryProgress) ProcessResponse(ctx context.Context, response *Response) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("ovm.sdp.response", protojson.Format(response)))

	// do not deal with responses that do not have a responder UUID
	ru, err := uuid.FromBytes(response.GetResponderUUID())
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("response", response).Error("Error parsing responder UUID")
		return
	}

	// Update the stored data
	qp.respondersMutex.Lock()
	defer qp.respondersMutex.Unlock()

	responder, exists := qp.responders[ru]

	if exists {
		responder.CancelMonitor()

		// Protect against out-of order responses. Do not mark a responder as
		// working if it has already finished
		if responder.lastState == ResponderState_COMPLETE ||
			responder.lastState == ResponderState_ERROR ||
			responder.lastState == ResponderState_CANCELLED {
			return
		}
	} else {
		// If the responder is new, add it to the list
		responder = &responderStatus{
			Name: response.GetResponder(),
			ID:   ru,
		}
		qp.responders[ru] = responder
	}

	responder.SetState(response.GetState())

	// Check if we should expect another response
	expectFollowUp := (response.GetNextUpdateIn() != nil && response.GetState() != ResponderState_COMPLETE)

	// If we are told to expect a new response, set up context for it
	if expectFollowUp {
		timeout := response.GetNextUpdateIn().AsDuration()

		monitorContext, monitorCancel := context.WithCancel(context.Background())

		responder.SetMonitorContext(monitorContext, monitorCancel) //nolint: contextcheck // we expect a new response

		// Create a goroutine to watch for a stalled connection
		go stallMonitor(monitorContext, timeout, responder, qp) //nolint: contextcheck // we expect a new response
	}

	// Finally check to see if this was the final request and if so drain
	// everything. We also need to check that the start timeout has elapsed to
	// ensure that we don't drain too early. The start timeout goroutine will
	// drain everything when it elapses if required
	if qp.allDone() && qp.StartTimeoutElapsed.Load() {
		// at this point I need to add some slack in case the we have received
		// the completion response before the final item. The sources are
		// supposed to wait until all items have been sent in order to send
		// this, but NATS doesn't guarantee ordering so there's still a
		// reasonable chance that things will arrive in a weird order. This is a
		// pretty bad solution and realistically this should be addressed in the
		// protocol itself, but for now this will do. Especially since it
		// doesn't actually block anything that the client sees, it's just
		// delaying cleanup for a little longer than we need
		time.Sleep(qp.DrainDelay)

		qp.Drain()
	}
}

// NumWorking returns the number of responders that are in the Working state
func (qp *QueryProgress) NumWorking() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numWorking()
}

// numWorking Returns the number of responders that are working without taking a
// lock
func (qp *QueryProgress) numWorking() int {
	var numWorking int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_WORKING {
			numWorking++
		}
	}

	return numWorking
}

// NumStalled returns the number of responders that are in the STALLED state
func (qp *QueryProgress) NumStalled() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numStalled()
}

// numStalled Returns the number of responders that are stalled without taking a
// lock
func (qp *QueryProgress) numStalled() int {
	var numStalled int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_STALLED {
			numStalled++
		}
	}

	return numStalled
}

// NumComplete returns the number of responders that are in the COMPLETE state
func (qp *QueryProgress) NumComplete() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numComplete()
}

// numComplete Returns the number of responders that are complete without taking
// a lock
func (qp *QueryProgress) numComplete() int {
	var numComplete int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_COMPLETE {
			numComplete++
		}
	}

	return numComplete
}

// NumError returns the number of responders that are in the ERROR state
func (qp *QueryProgress) NumError() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numError()
}

// numError Returns the number of responders that are in the ERROR state
// without taking a lock
func (qp *QueryProgress) numError() int {
	var numError int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_ERROR {
			numError++
		}
	}

	return numError
}

// NumCancelled returns the number of responders that are in the CANCELLED state
func (qp *QueryProgress) NumCancelled() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numCancelled()
}

// numCancelled Returns the number of responders that are in the CANCELLED state
// without taking a lock
func (qp *QueryProgress) numCancelled() int {
	var numCancelled int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_CANCELLED {
			numCancelled++
		}
	}

	return numCancelled
}

// NumResponders returns the total number of unique responders
func (qp *QueryProgress) NumResponders() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return qp.numResponders()
}

// numResponders Returns the total number of unique responders without taking a
// lock
func (qp *QueryProgress) numResponders() int {
	return len(qp.responders)
}

// ResponderStates Returns the status details for all responders as a map.
// Where the key is the name of the responder and the value is its status
func (qp *QueryProgress) ResponderStates() map[uuid.UUID]ResponderState {
	statuses := make(map[uuid.UUID]ResponderState)
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	for _, responder := range qp.responders {
		statuses[responder.ID] = responder.LastState()
	}

	return statuses
}

func (qp *QueryProgress) String() string {
	return fmt.Sprintf(
		"Working: %v\nStalled: %v\nComplete: %v\nFailed: %v\nCancelled: %v\nResponders: %v\n",
		qp.NumWorking(),
		qp.NumStalled(),
		qp.NumComplete(),
		qp.NumError(),
		qp.NumCancelled(),
		qp.NumResponders(),
	)
}

// Complete will return true if there are no remaining responders working, does
// not take locks
func (qp *QueryProgress) allDone() bool {
	if qp.numResponders() > 0 || qp.cancelled {
		// If we have had at least one response, and there aren't any waiting
		// then we are going to assume that everything is done. It is of course
		// possible that there has just been a very fast responder and so a
		// minimum execution time might be a good idea
		return (qp.numWorking() == 0)
	}
	// If there have been no responders at all we can't say that we're "done"
	return false
}

// stallMonitor watches for stalled connections. It should be passed the
// responder to monitor, the time to wait before marking the connection as
// stalled, and a context. The context is used to allow cancellation of the
// stall monitor from another thread in the case that another message is
// received.
func stallMonitor(ctx context.Context, timeout time.Duration, responder *responderStatus, qp *QueryProgress) {
	defer tracing.LogRecoverToReturn(ctx, "stallMonitor")
	select {
	case <-ctx.Done():
		// If the context is cancelled then we don't want to do anything
		return
	case <-time.After(timeout):
		// If the timeout elapses before the context is cancelled it
		// means that we haven't received a response in the expected
		// time, we now need to mark that responder as STALLED
		responder.SetState(ResponderState_STALLED)
		log.WithContext(ctx).WithField("ovm.timeout", timeout).WithField("ovm.responder", responder.Name).Error("marking responder as stalled after timeout")

		if qp.allDone() {
			qp.Drain()
		}

		return
	}
}

// unsubscribeGracefully Closes a NATS subscription gracefully, this includes
// draining, unsubscribing and ensuring that all callbacks are complete
func unsubscribeGracefully(s *nats.Subscription) error {
	if s != nil {
		// Drain NATS connections
		err := s.Drain()

		if err != nil {
			// If that fails, fall back to an unsubscribe
			err = s.Unsubscribe()

			if err != nil {
				return err
			}
		}

		// Wait for all items to finish processing, including all callbacks
	}

	return nil
}
