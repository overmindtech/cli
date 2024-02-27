package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// getSnapshotCmd represents the get-snapshot command
var getSnapshotCmd = &cobra.Command{
	Use:   "get-snapshot --uuid ID",
	Short: "Displays the contents of a snapshot.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-snapshot` flags")
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

		exitcode := GetSnapshot(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetSnapshot(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	snapshotUuid, err := uuid.Parse(viper.GetString("uuid"))
	if err != nil {
		log.Errorf("invalid --uuid value '%v', error: %v", viper.GetString("uuid"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI GetSnapshot", trace.WithAttributes(
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

	ctx, err = ensureToken(ctx, oi, []string{"changes:read"})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedSnapshotsClient(ctx, oi)
	response, err := client.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
		Msg: &sdp.GetSnapshotRequest{
			UUID: snapshotUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get snapshot")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"snapshot-uuid":        uuid.UUID(response.Msg.GetSnapshot().GetMetadata().GetUUID()),
		"snapshot-created":     response.Msg.GetSnapshot().GetMetadata().GetCreated().AsTime(),
		"snapshot-name":        response.Msg.GetSnapshot().GetProperties().GetName(),
		"snapshot-description": response.Msg.GetSnapshot().GetProperties().GetDescription(),
	}).Info("found snapshot")
	for _, q := range response.Msg.GetSnapshot().GetProperties().GetQueries() {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-query": q,
		}).Info("found snapshot query")
	}
	for _, i := range response.Msg.GetSnapshot().GetProperties().GetExcludedItems() {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-excluded-item": i,
		}).Info("found snapshot excluded item")
	}
	for _, i := range response.Msg.GetSnapshot().GetProperties().GetItems() {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-item": i,
		}).Info("found snapshot item")
	}

	b, err := json.MarshalIndent(response.Msg.GetSnapshot().ToMap(), "", "  ")
	if err != nil {
		log.Infof("Error rendering snapshot: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return 0
}

func init() {
	snapshotsCmd.AddCommand(getSnapshotCmd)

	getSnapshotCmd.PersistentFlags().String("uuid", "", "The UUID of the snapshot that should be displayed.")
}
