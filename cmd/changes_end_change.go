package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// endChangeCmd represents the end-change command
var endChangeCmd = &cobra.Command{
	Use:   "end-change --uuid ID",
	Short: "Finishes the specified change. Call this just after you finished the change. This will store a snapshot of the current system state for later reference.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `end-change` flags")
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

		exitcode := EndChange(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func EndChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI EndChange", trace.WithAttributes(
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

	ctx, _, err = ensureToken(ctx, oi, []string{"changes:write"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_HAPPENING, true)
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	// snapClient := AuthenticatedSnapshotsClient(ctx)
	client := AuthenticatedChangesClient(ctx, oi)
	stream, err := client.EndChange(ctx, &connect.Request[sdp.EndChangeRequest]{
		Msg: &sdp.EndChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to end change")
		return 1
	}
	log.WithContext(ctx).WithFields(lf).Info("processing")
	for stream.Receive() {
		msg := stream.Msg()
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"state": msg.GetState(),
			"items": msg.GetNumItems(),
			"edges": msg.GetNumEdges(),
		}).Info("progress")
	}
	if stream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(stream.Err()).Error("failed to process end change")
		return 1
	}

	log.WithContext(ctx).WithFields(lf).Info("finished change")
	return 0
}

func init() {
	changesCmd.AddCommand(endChangeCmd)

	addChangeUuidFlags(endChangeCmd)
}
