package sdpws

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
)

// GatewayMessageHandler is an interface that can be implemented to handle
// messages from the gateway. The individual methods are called when the sdpws
// client receives a message from the gateway. Methods are called in the same
// order as the messages are received from the gateway. The sdpws client
// guarantees that the methods are called in a single thread, so no locking is
// needed.
type GatewayMessageHandler interface {
	NewItem(context.Context, *sdp.Item)
	NewEdge(context.Context, *sdp.Edge)
	Status(context.Context, *sdp.GatewayRequestStatus)
	Error(context.Context, string)
	QueryError(context.Context, *sdp.QueryError)
	DeleteItem(context.Context, *sdp.Reference)
	DeleteEdge(context.Context, *sdp.Edge)
	UpdateItem(context.Context, *sdp.Item)
	SnapshotStoreResult(context.Context, *sdp.SnapshotStoreResult)
	SnapshotLoadResult(context.Context, *sdp.SnapshotLoadResult)
	BookmarkStoreResult(context.Context, *sdp.BookmarkStoreResult)
	BookmarkLoadResult(context.Context, *sdp.BookmarkLoadResult)
	QueryStatus(context.Context, *sdp.QueryStatus)
	ChatResponse(context.Context, *sdp.ChatResponse)
	ToolStart(context.Context, *sdp.ToolStart)
	ToolFinish(context.Context, *sdp.ToolFinish)
}

type LoggingGatewayMessageHandler struct {
	Level log.Level
}

// assert that LoggingGatewayMessageHandler implements GatewayMessageHandler
var _ GatewayMessageHandler = (*LoggingGatewayMessageHandler)(nil)

func (l *LoggingGatewayMessageHandler) NewItem(ctx context.Context, item *sdp.Item) {
	log.WithContext(ctx).WithField("item", item).Log(l.Level, "received new item")
}

func (l *LoggingGatewayMessageHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
	log.WithContext(ctx).WithField("edge", edge).Log(l.Level, "received new edge")
}

func (l *LoggingGatewayMessageHandler) Status(ctx context.Context, status *sdp.GatewayRequestStatus) {
	log.WithContext(ctx).WithField("status", status.GetSummary()).Log(l.Level, "received status")
}

func (l *LoggingGatewayMessageHandler) Error(ctx context.Context, errorMessage string) {
	log.WithContext(ctx).WithField("errorMessage", errorMessage).Log(l.Level, "received error")
}

func (l *LoggingGatewayMessageHandler) QueryError(ctx context.Context, queryError *sdp.QueryError) {
	log.WithContext(ctx).WithField("queryError", queryError).Log(l.Level, "received query error")
}

func (l *LoggingGatewayMessageHandler) DeleteItem(ctx context.Context, reference *sdp.Reference) {
	log.WithContext(ctx).WithField("reference", reference).Log(l.Level, "received delete item")
}

func (l *LoggingGatewayMessageHandler) DeleteEdge(ctx context.Context, edge *sdp.Edge) {
	log.WithContext(ctx).WithField("edge", edge).Log(l.Level, "received delete edge")
}

func (l *LoggingGatewayMessageHandler) UpdateItem(ctx context.Context, item *sdp.Item) {
	log.WithContext(ctx).WithField("item", item).Log(l.Level, "received updated item")
}

func (l *LoggingGatewayMessageHandler) SnapshotStoreResult(ctx context.Context, result *sdp.SnapshotStoreResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received snapshot store result")
}

func (l *LoggingGatewayMessageHandler) SnapshotLoadResult(ctx context.Context, result *sdp.SnapshotLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received snapshot load result")
}

func (l *LoggingGatewayMessageHandler) BookmarkStoreResult(ctx context.Context, result *sdp.BookmarkStoreResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received bookmark store result")
}

func (l *LoggingGatewayMessageHandler) BookmarkLoadResult(ctx context.Context, result *sdp.BookmarkLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received bookmark load result")
}

func (l *LoggingGatewayMessageHandler) QueryStatus(ctx context.Context, status *sdp.QueryStatus) {
	log.WithContext(ctx).WithField("status", status).WithField("uuid", status.GetUUIDParsed()).Log(l.Level, "received query status")
}

func (l *LoggingGatewayMessageHandler) ChatResponse(ctx context.Context, chatResponse *sdp.ChatResponse) {
	log.WithContext(ctx).WithField("chatResponse", chatResponse).Log(l.Level, "received chat response")
}

func (l *LoggingGatewayMessageHandler) ToolStart(ctx context.Context, toolStart *sdp.ToolStart) {
	log.WithContext(ctx).WithField("toolStart", toolStart).Log(l.Level, "received tool start")
}

func (l *LoggingGatewayMessageHandler) ToolFinish(ctx context.Context, toolFinish *sdp.ToolFinish) {
	log.WithContext(ctx).WithField("toolFinish", toolFinish).Log(l.Level, "received tool finish")
}

type NoopGatewayMessageHandler struct{}

// assert that NoopGatewayMessageHandler implements GatewayMessageHandler
var _ GatewayMessageHandler = (*NoopGatewayMessageHandler)(nil)

func (l *NoopGatewayMessageHandler) NewItem(ctx context.Context, item *sdp.Item) {
}

func (l *NoopGatewayMessageHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
}

func (l *NoopGatewayMessageHandler) Status(ctx context.Context, status *sdp.GatewayRequestStatus) {
}

func (l *NoopGatewayMessageHandler) Error(ctx context.Context, errorMessage string) {
}

func (l *NoopGatewayMessageHandler) QueryError(ctx context.Context, queryError *sdp.QueryError) {
}

func (l *NoopGatewayMessageHandler) DeleteItem(ctx context.Context, reference *sdp.Reference) {
}

func (l *NoopGatewayMessageHandler) DeleteEdge(ctx context.Context, edge *sdp.Edge) {
}

func (l *NoopGatewayMessageHandler) UpdateItem(ctx context.Context, item *sdp.Item) {
}

func (l *NoopGatewayMessageHandler) SnapshotStoreResult(ctx context.Context, result *sdp.SnapshotStoreResult) {
}

func (l *NoopGatewayMessageHandler) SnapshotLoadResult(ctx context.Context, result *sdp.SnapshotLoadResult) {
}

func (l *NoopGatewayMessageHandler) BookmarkStoreResult(ctx context.Context, result *sdp.BookmarkStoreResult) {
}

func (l *NoopGatewayMessageHandler) BookmarkLoadResult(ctx context.Context, result *sdp.BookmarkLoadResult) {
}

func (l *NoopGatewayMessageHandler) QueryStatus(ctx context.Context, status *sdp.QueryStatus) {
}

func (l *NoopGatewayMessageHandler) ChatResponse(ctx context.Context, chatMessageResult *sdp.ChatResponse) {
}

func (l *NoopGatewayMessageHandler) ToolStart(ctx context.Context, toolStart *sdp.ToolStart) {
}

func (l *NoopGatewayMessageHandler) ToolFinish(ctx context.Context, toolFinish *sdp.ToolFinish) {
}

var _ GatewayMessageHandler = (*StoreEverythingHandler)(nil)

// A handler that stores all the items and edges it receives
type StoreEverythingHandler struct {
	Items []*sdp.Item
	Edges []*sdp.Edge

	NoopGatewayMessageHandler
}

func (s *StoreEverythingHandler) NewItem(ctx context.Context, item *sdp.Item) {
	s.Items = append(s.Items, item)
}

func (s *StoreEverythingHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
	s.Edges = append(s.Edges, edge)
}

var _ GatewayMessageHandler = (*WaitForAllQueriesHandler)(nil)

// A Handler that waits for all queries to be done then calls a callback
type WaitForAllQueriesHandler struct {
	// A callback that will be called when all queries are done
	DoneCallback func()

	StoreEverythingHandler
}

func (w *WaitForAllQueriesHandler) Status(ctx context.Context, status *sdp.GatewayRequestStatus) {
	if status.Done() {
		w.DoneCallback()
	}
}
