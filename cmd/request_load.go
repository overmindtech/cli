package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// requestLoadCmd represents the start command
var requestLoadCmd = &cobra.Command{
	Use:    "load",
	Short:  "Loads a snapshot or bookmark from the overmind API",
	PreRun: PreRunSetup,
	RunE:   RequestLoad,
}

func RequestLoad(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var uuidString string
	var u uuid.UUID

	isBookmark := false

	if viper.GetString("bookmark-uuid") != "" {
		uuidString = viper.GetString("bookmark-uuid")
		isBookmark = true
	} else if viper.GetString("snapshot-uuid") != "" {
		uuidString = viper.GetString("snapshot-uuid")
	} else {
		return flagError{fmt.Sprintf("No bookmark or snapshot UUID provided\n\n%v", cmd.UsageString())}
	}

	u, err := uuid.Parse(uuidString)
	if err != nil {
		return flagError{fmt.Sprintf("Failed to parse UUID '%v': %v\n\n%v", uuidString, err, cmd.UsageString())}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"explore:read", "changes:read"}, nil)
	if err != nil {
		return err
	}

	lf := log.Fields{
		"uuid": u,
	}

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
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to connect to overmind API",
		}
	}
	defer c.Close(ctx)

	// Send the load request
	if isBookmark {
		err = c.SendLoadBookmark(ctx, &sdp.LoadBookmark{
			UUID: u[:],
		})
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to send load bookmark request",
			}
		}

		result, err := handler.WaitBookmarkResult(ctx)
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to receive for bookmark result",
			}
		}

		log.WithContext(ctx).WithFields(lf).WithField("result", result).Info("bookmark loaded")
	} else if viper.GetString("snapshot-uuid") != "" {
		err = c.SendLoadSnapshot(ctx, &sdp.LoadSnapshot{
			UUID: u[:],
		})
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to send load snapshot request",
			}
		}

		result, err := handler.WaitSnapshotResult(ctx)
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to receive for snapshot result",
			}
		}

		log.WithContext(ctx).WithFields(lf).WithField("result", result).Info("snapshot loaded")
	}

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

func init() {
	requestCmd.AddCommand(requestLoadCmd)

	addAPIFlags(requestLoadCmd)

	requestLoadCmd.PersistentFlags().String("dump-json", "", "Dump the request to the given file as JSON")

	requestLoadCmd.PersistentFlags().String("bookmark-uuid", "", "The UUID of the bookmark or snapshot to load")
	requestLoadCmd.PersistentFlags().String("snapshot-uuid", "", "The UUID of the snapshot to load")
}
