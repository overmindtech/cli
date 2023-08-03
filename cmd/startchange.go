package cmd

import (
	"context"
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

// startChangeCmd represents the start-change command
var startChangeCmd = &cobra.Command{
	Use:   "start-change --uuid ID",
	Short: "Starts the specified change. Call this just before you're about to start the change. This will store a snapshot of the current system state for later reference.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `start-change` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := StartChange(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func StartChange(signals chan os.Signal, ready chan bool) int {
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
	ctx, span := tracing.Tracer().Start(ctx, "CLI StartChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	lf := log.Fields{
		"uuid": snapshotUuid.String(),
	}

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// snapClient := AuthenticatedSnapshotsClient(ctx)
	client := AuthenticatedChangesClient(ctx)
	stream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
		Msg: &sdp.StartChangeRequest{
			ChangeUUID: snapshotUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to start change")
		return 1
	}
	log.WithContext(ctx).WithFields(lf).Info("processing")
	for stream.Receive() {
		msg := stream.Msg()
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"state": msg.State,
			"items": msg.NumItems,
			"edges": msg.NumEdges,
		}).Info("progress")
	}
	log.WithContext(ctx).WithFields(lf).Info("started change")
	return 0
}

func init() {
	rootCmd.AddCommand(startChangeCmd)

	startChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")

	startChangeCmd.PersistentFlags().String("uuid", "", "The UUID of the snapshot that should be displayed.")

	startChangeCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
