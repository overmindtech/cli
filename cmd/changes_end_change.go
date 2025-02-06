package cmd

import (
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// endChangeCmd represents the end-change command
var endChangeCmd = &cobra.Command{
	Use:    "end-change --uuid ID",
	Short:  "Finishes the specified change. Call this just after you finished the change. This will store a snapshot of the current system state for later reference.",
	PreRun: PreRunSetup,
	RunE:   EndChange,
}

func EndChange(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"}, nil)
	if err != nil {
		return err
	}

	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_HAPPENING, viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
	}

	lf := log.Fields{"uuid": changeUuid.String()}

	client := AuthenticatedChangesClient(ctx, oi)
	stream, err := client.EndChange(ctx, &connect.Request[sdp.EndChangeRequest]{
		Msg: &sdp.EndChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to end change",
		}
	}
	log.WithContext(ctx).WithFields(lf).Info("processing")
	lastLog := time.Now().Add(-1 * time.Minute)
	for stream.Receive() {
		msg := stream.Msg()
		// print progress every 2 seconds
		if time.Now().After(lastLog.Add(2 * time.Second)) {
			log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
				"state": msg.GetState(),
				"items": msg.GetNumItems(),
				"edges": msg.GetNumEdges(),
			}).Info("progress")
			lastLog = time.Now()
		}
	}
	if stream.Err() != nil {
		return loggedError{
			err:     stream.Err(),
			fields:  lf,
			message: "failed to process end change",
		}
	}

	log.WithContext(ctx).WithFields(lf).Info("finished change")
	return nil
}

func init() {
	changesCmd.AddCommand(endChangeCmd)

	addChangeUuidFlags(endChangeCmd)
}
