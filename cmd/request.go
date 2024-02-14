package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/internal"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goroutine to watch for cancellation signals
		go func() {
			select {
			case <-sigs:
				cancel()
			case <-ctx.Done():
			}
		}()

		exitcode := Request(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

// requestHandler is a simple implementation of GatewayMessageHandler that
// implements the required logging for the `request` command.
type requestHandler struct {
	lf log.Fields

	queriesStarted int

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
		log.WithContext(ctx).WithFields(l.lf).WithFields(statusFields).Debugf("Received QueryStatus with nil UUID")
		return
	}
	statusFields["query"] = queryUuid

	if status.GetStatus() == sdp.QueryStatus_STARTED {
		l.queriesStarted += 1
	}

	// nolint:exhaustive // we _want_ to log all other status fields as unexpected
	switch status.GetStatus() {
	case sdp.QueryStatus_STARTED, sdp.QueryStatus_FINISHED, sdp.QueryStatus_ERRORED, sdp.QueryStatus_CANCELLED:
		// do nothing
	default:
		statusFields["unexpected_status"] = true
	}

	log.WithContext(ctx).WithFields(l.lf).WithFields(statusFields).Debugf("query status update")
}

func Request(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI Request", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	lf := log.Fields{}

	ctx, err = ensureToken(ctx, []string{"explore:read"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	handler := &requestHandler{
		lf:                           lf,
		LoggingGatewayMessageHandler: sdpws.LoggingGatewayMessageHandler{Level: log.TraceLevel},
		items:                        []*sdp.Item{},
		edges:                        []*sdp.Edge{},
		msgLog:                       []*sdp.GatewayResponse{},
	}
	gatewayUrl := internal.GatewayURL(viper.GetString("url"))
	c, err := sdpws.DialBatch(ctx, gatewayUrl,
		NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
		handler,
	)
	if err != nil {
		lf["gateway-url"] = gatewayUrl
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to overmind API")
		return 1
	}
	defer c.Close(ctx)

	q, err := createQuery()
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to create query")
		return 1
	}
	err = c.SendQuery(ctx, q)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to execute query")
		return 1
	}
	log.WithContext(ctx).WithFields(lf).WithError(err).Info("received items")

	// Log the request in JSON
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to marshal query for logging")
		return 1
	}
	log.WithContext(ctx).WithFields(lf).WithField("uuid", uuid.UUID(q.GetUUID())).Infof("Query:\n%v", string(b))

	err = c.Wait(ctx, uuid.UUIDs{uuid.UUID(q.GetUUID())})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("queries failed")
	}

	log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
		"queriesStarted": handler.queriesStarted,
		"itemsReceived":  len(handler.items),
		"edgesReceived":  len(handler.edges),
	}).Info("all queries done")

	dumpFileName := viper.GetString("dump-json")
	if dumpFileName != "" {
		f, err := os.Create(dumpFileName)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithField("file", dumpFileName).WithError(err).Error("Failed to open file for dumping")
			return 1
		}
		defer f.Close()
		type dump struct {
			Msgs []*sdp.GatewayResponse `json:"msgs"`
		}
		err = json.NewEncoder(f).Encode(dump{
			Msgs: handler.msgLog,
		})
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithField("file", dumpFileName).WithError(err).Error("Failed to dump to file")
			return 1
		}
		log.WithContext(ctx).WithFields(lf).WithField("file", dumpFileName).Info("dumped to file")
	}

	if viper.GetBool("snapshot-after") {
		log.WithContext(ctx).Info("Starting snapshot")
		snId, err := c.StoreSnapshot(ctx, viper.GetString("snapshot-name"), viper.GetString("snapshot-description"))
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to send snapshot request")
			return 1
		}

		log.WithContext(ctx).WithFields(lf).Infof("Snapshot stored successfully: %v", snId)
		return 0
	}

	return 0
}

func methodFromString(method string) (sdp.QueryMethod, error) {
	var result sdp.QueryMethod

	switch method {
	case "get":
		result = sdp.QueryMethod_GET
	case "list":
		result = sdp.QueryMethod_LIST
	case "search":
		result = sdp.QueryMethod_SEARCH
	default:
		return 0, fmt.Errorf("query method '%v' not supported", method)
	}
	return result, nil
}

func createQuery() (*sdp.Query, error) {
	u := uuid.New()
	method, err := methodFromString(viper.GetString("query-method"))
	if err != nil {
		return nil, err
	}

	return &sdp.Query{
		Method:   method,
		Type:     viper.GetString("query-type"),
		Query:    viper.GetString("query"),
		Scope:    viper.GetString("query-scope"),
		Deadline: timestamppb.New(time.Now().Add(10 * time.Hour)),
		UUID:     u[:],
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth:                  viper.GetUint32("link-depth"),
			FollowOnlyBlastPropagation: viper.GetBool("blast-radius"),
		},
		IgnoreCache: viper.GetBool("ignore-cache"),
	}, nil
}

func init() {
	rootCmd.AddCommand(requestCmd)

	requestCmd.PersistentFlags().String("request-type", "query", "The type of request to send (query, load-bookmark, load-snapshot)")
	requestCmd.PersistentFlags().String("dump-json", "", "Dump the request to the given file as JSON")

	requestCmd.PersistentFlags().String("query-method", "get", "The method to use (get, list, search)")
	requestCmd.PersistentFlags().String("query-type", "*", "The type to query")
	requestCmd.PersistentFlags().String("query", "", "The actual query to send")
	requestCmd.PersistentFlags().String("query-scope", "*", "The scope to query")
	requestCmd.PersistentFlags().Bool("ignore-cache", false, "Set to true to ignore all caches in overmind.")

	requestCmd.PersistentFlags().String("bookmark-uuid", "", "The UUID of the bookmark to load")
	requestCmd.PersistentFlags().String("snapshot-uuid", "", "The UUID of the snapshot to load")

	requestCmd.PersistentFlags().Bool("snapshot-after", false, "Set this to create a snapshot of the query results")
	requestCmd.PersistentFlags().String("snapshot-name", "CLI", "The snapshot name of the query results")
	requestCmd.PersistentFlags().String("snapshot-description", "none", "The snapshot description of the query results")

	requestCmd.PersistentFlags().String("timeout", "5m", "How long to wait for responses")
	requestCmd.PersistentFlags().Uint32("link-depth", 0, "How deeply to link")
	requestCmd.PersistentFlags().Bool("blast-radius", false, "Whether to query using blast radius, note that if using this option, link-depth should be set to > 0")
}
