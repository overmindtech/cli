package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// requestCmd represents the start command
var requestCmd = &cobra.Command{
	Use:     "request",
	GroupID: "api",
	Short:   "Runs a request against the overmind API",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `request` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// requestHandler is a simple implementation of GatewayMessageHandler that
// implements the required logging for the `request` command.
type requestHandler struct {
	lf log.Fields

	queriesStarted int

	snapshotLoadResult chan *sdp.SnapshotLoadResult
	bookmarkLoadResult chan *sdp.BookmarkLoadResult

	items  []*sdp.Item
	edges  []*sdp.Edge
	msgLog []*sdp.GatewayResponse

	sdpws.LoggingGatewayMessageHandler
}

// assert that requestHandler implements GatewayMessageHandler
var _ sdpws.GatewayMessageHandler = (*requestHandler)(nil)

func (l *requestHandler) NewItem(ctx context.Context, item *sdp.Item) {
	l.LoggingGatewayMessageHandler.NewItem(ctx, item)
	l.items = append(l.items, item)
	l.msgLog = append(l.msgLog, &sdp.GatewayResponse{
		ResponseType: &sdp.GatewayResponse_NewItem{NewItem: item},
	})
	log.WithContext(ctx).WithFields(l.lf).WithField("item", item.GloballyUniqueName()).Infof("new item")
}

func (l *requestHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
	l.LoggingGatewayMessageHandler.NewEdge(ctx, edge)
	l.edges = append(l.edges, edge)
	l.msgLog = append(l.msgLog, &sdp.GatewayResponse{
		ResponseType: &sdp.GatewayResponse_NewEdge{NewEdge: edge},
	})
	log.WithContext(ctx).WithFields(l.lf).WithFields(log.Fields{
		"from": edge.GetFrom().GloballyUniqueName(),
		"to":   edge.GetTo().GloballyUniqueName(),
	}).Info("new edge")
}

func (l *requestHandler) Error(ctx context.Context, errorMessage string) {
	log.WithContext(ctx).WithFields(l.lf).Errorf("generic error: %v", errorMessage)
}

func (l *requestHandler) QueryError(ctx context.Context, err *sdp.QueryError) {
	log.WithContext(ctx).WithFields(l.lf).Errorf("Error for %v from %v(%v): %v", uuid.Must(uuid.FromBytes(err.GetUUID())), err.GetResponderName(), err.GetSourceName(), err)
}

func (l *requestHandler) QueryStatus(ctx context.Context, status *sdp.QueryStatus) {
	l.LoggingGatewayMessageHandler.QueryStatus(ctx, status)
	statusFields := log.Fields{
		"status": status.GetStatus().String(),
	}
	queryUuid := status.GetUUIDParsed()
	if queryUuid == nil {
		log.WithContext(ctx).WithFields(l.lf).WithFields(statusFields).Debug("Received QueryStatus with nil UUID")
		return
	}
	statusFields["query"] = queryUuid

	if status.GetStatus() == sdp.QueryStatus_STARTED {
		l.queriesStarted += 1
	}

	//nolint:exhaustive // we _want_ to log all other status fields as unexpected
	switch status.GetStatus() {
	case sdp.QueryStatus_STARTED, sdp.QueryStatus_FINISHED, sdp.QueryStatus_ERRORED, sdp.QueryStatus_CANCELLED:
		// do nothing
	default:
		statusFields["unexpected_status"] = true
	}

	log.WithContext(ctx).WithFields(l.lf).WithFields(statusFields).Debug("query status update")
}

// Waits for the next snapshot load result to be received.
func (l *requestHandler) WaitSnapshotResult(ctx context.Context) (*sdp.SnapshotLoadResult, error) {
	select {
	case result := <-l.snapshotLoadResult:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Waits for the next bookmark load result to be received.
func (l *requestHandler) WaitBookmarkResult(ctx context.Context) (*sdp.BookmarkLoadResult, error) {
	select {
	case result := <-l.bookmarkLoadResult:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *requestHandler) SnapshotLoadResult(ctx context.Context, result *sdp.SnapshotLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received snapshot load result")
	l.snapshotLoadResult <- result
}

func (l *requestHandler) BookmarkLoadResult(ctx context.Context, result *sdp.BookmarkLoadResult) {
	log.WithContext(ctx).WithField("result", result).Log(l.Level, "received bookmark load result")
	l.bookmarkLoadResult <- result
}

func init() {
	rootCmd.AddCommand(requestCmd)

	addAPIFlags(requestCmd)

}
