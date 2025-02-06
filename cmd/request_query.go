package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// requestQueryCmd represents the start command
var requestQueryCmd = &cobra.Command{
	Use:    "query",
	Short:  "Runs an SDP query against the overmind API",
	PreRun: PreRunSetup,
	RunE:   RequestQuery,
}

func RequestQuery(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"explore:read", "changes:read"}, nil)
	if err != nil {
		return err
	}

	lf := log.Fields{}
	handler := &requestHandler{
		lf:                           lf,
		LoggingGatewayMessageHandler: sdpws.LoggingGatewayMessageHandler{Level: log.TraceLevel},
		items:                        []*sdp.Item{},
		edges:                        []*sdp.Edge{},
		msgLog:                       []*sdp.GatewayResponse{},
		bookmarkLoadResult:           make(chan *sdp.BookmarkLoadResult, 128),
		snapshotLoadResult:           make(chan *sdp.SnapshotLoadResult, 128),
	}
	gatewayUrl := oi.GatewayUrl()
	lf["gateway-url"] = gatewayUrl
	c, err := sdpws.DialBatch(ctx, gatewayUrl,
		NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
		handler,
	)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to overmind API")
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to connect to overmind API",
		}
	}
	defer c.Close(ctx)

	q, err := createQuery()
	if err != nil {
		return flagError{usage: fmt.Sprintf("invalid query: %v\n\n%v", err, cmd.UsageString())}
	}
	err = c.SendQuery(ctx, q)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to execute query",
		}
	}
	log.WithContext(ctx).WithFields(lf).WithError(err).Info("received items")

	// Log the request in JSON
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to marshal query for logging",
		}
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
			lf["file"] = dumpFileName
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to open file for dumping",
			}
		}
		defer f.Close()
		type dump struct {
			Msgs []*sdp.GatewayResponse `json:"msgs"`
		}
		err = json.NewEncoder(f).Encode(dump{
			Msgs: handler.msgLog,
		})
		if err != nil {
			lf["file"] = dumpFileName
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to dump to file",
			}
		}
		log.WithContext(ctx).WithFields(lf).WithField("file", dumpFileName).Info("dumped to file")
	}

	if viper.GetBool("snapshot-after") {
		log.WithContext(ctx).Info("Starting snapshot")
		snId, err := c.StoreSnapshot(ctx, viper.GetString("snapshot-name"), viper.GetString("snapshot-description"))
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to send snapshot request",
			}
		}

		log.WithContext(ctx).WithFields(lf).Infof("Snapshot stored successfully: %v", snId)
	}

	return nil
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
