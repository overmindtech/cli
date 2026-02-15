package cmd

import (
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/workspace/sdp-go"
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

	// Resolve the change UUID without checking status. The server-side
	// EndChangeSimple handles status validation atomically and queues end-change
	// behind start-change if needed, avoiding the TOCTOU race where status
	// transitions between client-side checks.
	changeUuid, err := getChangeUUID(ctx, oi, viper.GetString("ticket-link"))
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
	}

	lf := log.Fields{"uuid": changeUuid.String()}

	// Call the simple RPC (enqueues a background job and returns immediately)
	client := AuthenticatedChangesClient(ctx, oi)
	resp, err := client.EndChangeSimple(ctx, &connect.Request[sdp.EndChangeRequest]{
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

	queuedAfterStart := resp.Msg.GetQueuedAfterStart()
	waitForSnapshot := viper.GetBool("wait-for-snapshot")
	if waitForSnapshot {
		// Poll until change status is DONE
		log.WithContext(ctx).WithFields(lf).Info("waiting for snapshot to complete")
		for {
			changeResp, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
				Msg: &sdp.GetChangeRequest{
					UUID: changeUuid[:],
				},
			})
			if err != nil {
				return loggedError{
					err:     err,
					fields:  lf,
					message: "failed to get change status",
				}
			}
			if changeResp.Msg.GetChange().GetMetadata().GetStatus() == sdp.ChangeStatus_CHANGE_STATUS_DONE {
				break
			}
			log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
				"status": changeResp.Msg.GetChange().GetMetadata().GetStatus().String(),
			}).Info("waiting for snapshot")
			time.Sleep(3 * time.Second)

			// check if the context is cancelled
			if ctx.Err() != nil {
				return loggedError{
					err:     ctx.Err(),
					fields:  lf,
					message: "context cancelled",
				}
			}
		}
		log.WithContext(ctx).WithFields(lf).Info("finished change")
	} else {
		if queuedAfterStart {
			log.WithContext(ctx).WithFields(lf).Info("change end queued (will run after start-change completes)")
		} else {
			log.WithContext(ctx).WithFields(lf).Info("change end initiated (processing in background)")
		}
	}
	return nil
}

func init() {
	changesCmd.AddCommand(endChangeCmd)

	addChangeUuidFlags(endChangeCmd)

	endChangeCmd.PersistentFlags().Bool("wait-for-snapshot", false, "Wait for the snapshot to complete before returning. Defaults to false.")
}
