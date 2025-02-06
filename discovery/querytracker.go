package discovery

import (
	"context"
	"errors"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// QueryTracker is used for tracking the progress of a single query. This
// is used because a single query could have a link depth that results in many
// additional queries being executed meaning that we need to not only track the first
// query, but also all other queries and items that result from linking
type QueryTracker struct {
	// The query to track
	Query *sdp.Query

	Context context.Context    // The context that this query is running in
	Cancel  context.CancelFunc // The cancel function for the context

	// The engine that this is connected to, used for sending NATS messages
	Engine *Engine
}

// Execute Executes a given item query and publishes results and errors on the
// relevant nats subjects. Returns the full list of items, errors, and a final
// error. The final error will be populated if all adapters failed, or some other
// error was encountered while trying run the query
//
// If the context is cancelled, all query work will stop
func (qt *QueryTracker) Execute(ctx context.Context) ([]*sdp.Item, []*sdp.QueryError, error) {
	if qt.Query == nil {
		return nil, nil, nil
	}

	if qt.Engine == nil {
		return nil, nil, errors.New("no engine supplied, cannot execute")
	}

	span := trace.SpanFromContext(ctx)

	items := make(chan *sdp.Item)
	errs := make(chan *sdp.QueryError)
	errChan := make(chan error)
	sdpErrs := make([]*sdp.QueryError, 0)
	sdpItems := make([]*sdp.Item, 0)

	// Run the query
	go func(e chan error) {
		defer tracing.LogRecoverToReturn(ctx, "Execute -> ExecuteQuery")
		e <- qt.Engine.ExecuteQuery(ctx, qt.Query, items, errs)
	}(errChan)

	// Process the items and errors as they come in
	for {
		select {
		case item, ok := <-items:
			if ok {
				sdpItems = append(sdpItems, item)

				if qt.Query.Subject() != "" && qt.Engine.natsConnection != nil {
					// Respond with the Item
					err := qt.Engine.natsConnection.Publish(ctx, qt.Query.Subject(), &sdp.QueryResponse{
						ResponseType: &sdp.QueryResponse_NewItem{
							NewItem: item,
						},
					})

					if err != nil {
						span.RecordError(err)
						log.WithFields(log.Fields{
							"error": err,
						}).Error("Response publishing error")
					}
				}
			} else {
				items = nil
			}
		case err, ok := <-errs:
			if ok {
				sdpErrs = append(sdpErrs, err)

				if qt.Query.Subject() != "" && qt.Engine.natsConnection != nil {
					pubErr := qt.Engine.natsConnection.Publish(ctx, qt.Query.Subject(), &sdp.QueryResponse{ResponseType: &sdp.QueryResponse_Error{Error: err}})

					if pubErr != nil {
						span.RecordError(err)
						log.WithFields(log.Fields{
							"error": err,
						}).Error("Error publishing item query error")
					}
				}
			} else {
				errs = nil
			}
		}

		if items == nil && errs == nil {
			// If both channels have been closed and set to nil, we're done so
			// break
			break
		}
	}

	// Get the result of the execution
	err := <-errChan

	if err != nil {
		return sdpItems, sdpErrs, err
	}

	return sdpItems, sdpErrs, ctx.Err()
}
