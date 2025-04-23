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
func (qt *QueryTracker) Execute(ctx context.Context) ([]*sdp.Item, []*sdp.Edge, []*sdp.QueryError, error) {
	if qt.Query == nil {
		return nil, nil, nil, nil
	}

	if qt.Engine == nil {
		return nil, nil, nil, errors.New("no engine supplied, cannot execute")
	}

	span := trace.SpanFromContext(ctx)

	responses := make(chan *sdp.QueryResponse)
	errChan := make(chan error, 1)

	sdpItems := make([]*sdp.Item, 0)
	sdpEdges := make([]*sdp.Edge, 0)
	sdpErrs := make([]*sdp.QueryError, 0)

	// Run the query in the background
	go func(e chan error) {
		defer tracing.LogRecoverToReturn(ctx, "Execute -> ExecuteQuery")
		defer close(e)
		e <- qt.Engine.ExecuteQuery(ctx, qt.Query, responses)
	}(errChan)

	// Process the responses as they come in
	for response := range responses {
		if qt.Query.Subject() != "" && qt.Engine.natsConnection != nil {
			err := qt.Engine.natsConnection.Publish(ctx, qt.Query.Subject(), response)
			if err != nil {
				span.RecordError(err)
				log.WithError(err).Error("Response publishing error")
			}
		}

		switch response := response.GetResponseType().(type) {
		case *sdp.QueryResponse_NewItem:
			sdpItems = append(sdpItems, response.NewItem)
		case *sdp.QueryResponse_Edge:
			sdpEdges = append(sdpEdges, response.Edge)
		case *sdp.QueryResponse_Error:
			sdpErrs = append(sdpErrs, response.Error)
		}
	}

	// Get the result of the execution
	err := <-errChan
	if err != nil {
		return sdpItems, sdpEdges, sdpErrs, err
	}

	return sdpItems, sdpEdges, sdpErrs, ctx.Err()
}
