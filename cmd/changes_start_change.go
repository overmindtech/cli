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

		var exitcode int
		// create a sub-scope to defer the span.End() to a point before shutting down the tracer
		func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctx, span := tracing.Tracer().Start(ctx, "CLI StartChange", trace.WithAttributes(
				attribute.String("ovm.config", fmt.Sprintf("%v", tracedSettings())),
			))
			defer span.End()

			// Create a goroutine to watch for cancellation signals
			go func() {
				select {
				case <-sigs:
					cancel()
				case <-ctx.Done():
				}
			}()

			exitcode = StartChange(ctx, nil)

			span.SetAttributes(attribute.Int("ovm.cli.exitcode", exitcode))
			if exitcode != 0 {
				span.SetAttributes(attribute.Bool("ovm.cli.fatalError", true))
			}
		}()

		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func StartChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

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

	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), true)
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	// snapClient := AuthenticatedSnapshotsClient(ctx)
	client := AuthenticatedChangesClient(ctx, oi)
	stream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
		Msg: &sdp.StartChangeRequest{
			ChangeUUID: changeUuid[:],
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
			"state": msg.GetState(),
			"items": msg.GetNumItems(),
			"edges": msg.GetNumEdges(),
		}).Info("progress")
	}
	if stream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(stream.Err()).Error("failed to process start change")
		return 1
	}

	log.WithContext(ctx).WithFields(lf).Info("started change")
	return 0
}

func init() {
	changesCmd.AddCommand(startChangeCmd)

	addChangeUuidFlags(startChangeCmd)
}
