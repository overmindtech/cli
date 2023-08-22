package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
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

		exitcode := GetSnapshot(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetSnapshot(signals chan os.Signal, ready chan bool) int {
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

	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI GetSnapshot", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, []string{"changes:read"}, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedSnapshotsClient(ctx)
	response, err := client.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
		Msg: &sdp.GetSnapshotRequest{
			UUID: snapshotUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"snapshot-url": viper.GetString("snapshot-url"),
		}).Error("failed to get snapshot")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"snapshot-uuid":        uuid.UUID(response.Msg.Snapshot.Metadata.UUID),
		"snapshot-created":     response.Msg.Snapshot.Metadata.Created.AsTime(),
		"snapshot-name":        response.Msg.Snapshot.Properties.Name,
		"snapshot-description": response.Msg.Snapshot.Properties.Description,
	}).Info("found snapshot")
	for _, q := range response.Msg.Snapshot.Properties.Queries {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-query": q,
		}).Info("found snapshot query")
	}
	for _, i := range response.Msg.Snapshot.Properties.ExcludedItems {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-excluded-item": i,
		}).Info("found snapshot excluded item")
	}
	for _, i := range response.Msg.Snapshot.Properties.Items {
		log.WithContext(ctx).WithFields(log.Fields{
			"snapshot-item": i,
		}).Info("found snapshot item")
	}

	b, err := json.MarshalIndent(response.Msg.Snapshot.ToMap(), "", "  ")
	if err != nil {
		log.Infof("Error rendering snapshot: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return 0
}

func init() {
	rootCmd.AddCommand(getSnapshotCmd)

	getSnapshotCmd.PersistentFlags().String("snapshot-url", "", "The snapshot service API endpoint (defaults to --url)")
	getSnapshotCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")

	getSnapshotCmd.PersistentFlags().String("uuid", "", "The UUID of the snapshot that should be displayed.")

	getSnapshotCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
