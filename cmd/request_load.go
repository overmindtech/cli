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
)

// requestLoadCmd represents the start command
var requestLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Loads a snapshot or bookmark from the overmind API",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `load` flags")
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

		exitcode := Load(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func Load(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI Request", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
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

	var uuidString string
	var u uuid.UUID

	if viper.GetString("bookmark-uuid") != "" {
		uuidString = viper.GetString("bookmark-uuid")
	} else if viper.GetString("snapshot-uuid") != "" {
		uuidString = viper.GetString("snapshot-uuid")
	} else {
		log.WithContext(ctx).WithFields(lf).Error("No bookmark or snapshot UUID provided")
		return 1
	}

	u, err = uuid.Parse(uuidString)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to parse UUID")
		return 1
	}

	// Send the load request
	if viper.GetString("bookmark-uuid") != "" {
		err = c.SendLoadBookmark(ctx, &sdp.LoadBookmark{
			UUID: u[:],
		})
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to send load bookmark request")
			return 1
		}

		result, err := handler.WaitBookmarkResult(ctx)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to wait for bookmark result")
			return 1
		}

		log.WithContext(ctx).WithFields(lf).WithField("result", result).Info("bookmark loaded")
	} else if viper.GetString("snapshot-uuid") != "" {
		err = c.SendLoadSnapshot(ctx, &sdp.LoadSnapshot{
			UUID: u[:],
		})
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to send load snapshot request")
			return 1
		}

		result, err := handler.WaitSnapshotResult(ctx)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to wait for snapshot result")
			return 1
		}

		log.WithContext(ctx).WithFields(lf).WithField("result", result).Info("snapshot loaded")
	} else {
		log.WithContext(ctx).WithFields(lf).Error("No bookmark or snapshot UUID provided")
		return 1
	}

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

func init() {
	requestCmd.AddCommand(requestLoadCmd)

	addAPIFlags(requestLoadCmd)

	requestLoadCmd.PersistentFlags().String("dump-json", "", "Dump the request to the given file as JSON")

	requestLoadCmd.PersistentFlags().String("bookmark-uuid", "", "The UUID of the bookmark or snapshot to load")
	requestLoadCmd.PersistentFlags().String("snapshot-uuid", "", "The UUID of the snapshot to load")
}
