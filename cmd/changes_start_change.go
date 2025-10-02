package cmd

import (
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startChangeCmd represents the start-change command
var startChangeCmd = &cobra.Command{
	Use:    "start-change --uuid ID",
	Short:  "Starts the specified change. Call this just before you're about to start the change. This will store a snapshot of the current system state for later reference.",
	PreRun: PreRunSetup,
	RunE:   StartChange,
}

func StartChange(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"}, nil)
	if err != nil {
		return err
	}

	changeUuid, err := getChangeUUIDAndCheckStatus(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err: err,
			fields: log.Fields{
				"ticket-link": viper.GetString("ticket-link"),
			},
			message: "failed to identify change",
		}
	}

	lf := log.Fields{
		"uuid":        changeUuid.String(),
		"ticket-link": viper.GetString("ticket-link"),
	}

	client := AuthenticatedChangesClient(ctx, oi)
	stream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
		Msg: &sdp.StartChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to start change",
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
			err:     err,
			fields:  lf,
			message: "failed to process start change",
		}
	}

	log.WithContext(ctx).WithFields(lf).Info("started change")
	return nil
}

func init() {
	changesCmd.AddCommand(startChangeCmd)

	addChangeUuidFlags(startChangeCmd)
}
