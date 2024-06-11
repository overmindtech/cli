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

// requestQueryCmd represents the start command
var requestQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Runs an SDP query against the overmind API",
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

		exitcode := Query(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func Query(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI Request", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", tracedSettings())),
	))
	defer span.End()

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get instance data from app")
		return 1
	}
	ctx, _, err = ensureToken(ctx, oi, []string{"explore:read"})
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
	gatewayUrl := oi.GatewayUrl()
	lf["gateway-url"] = gatewayUrl
	c, err := sdpws.DialBatch(ctx, gatewayUrl,
		NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
		handler,
	)
	if err != nil {
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
	requestCmd.AddCommand(requestQueryCmd)

	addAPIFlags(requestQueryCmd)

	requestQueryCmd.PersistentFlags().String("dump-json", "", "Dump the request to the given file as JSON")

	requestQueryCmd.PersistentFlags().String("query-method", "get", "The method to use (get, list, search)")
	requestQueryCmd.PersistentFlags().String("query-type", "*", "The type to query")
	requestQueryCmd.PersistentFlags().String("query", "", "The actual query to send")
	requestQueryCmd.PersistentFlags().String("query-scope", "*", "The scope to query")
	requestQueryCmd.PersistentFlags().Bool("ignore-cache", false, "Set to true to ignore all caches in overmind.")

	requestQueryCmd.PersistentFlags().Bool("snapshot-after", false, "Set this to create a snapshot of the query results")
	requestQueryCmd.PersistentFlags().String("snapshot-name", "CLI", "The snapshot name of the query results")
	requestQueryCmd.PersistentFlags().String("snapshot-description", "none", "The snapshot description of the query results")

	requestQueryCmd.PersistentFlags().Uint32("link-depth", 0, "How deeply to link")
	requestQueryCmd.PersistentFlags().Bool("blast-radius", false, "Whether to query using blast radius, note that if using this option, link-depth should be set to > 0")
}
