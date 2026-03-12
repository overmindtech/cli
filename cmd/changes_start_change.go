package cmd

import (
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/go/sdp-go"
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

	// wait for change analysis to complete (poll GetChange by change_analysis_status)
	client := AuthenticatedChangesClient(ctx, oi)
	if err := waitForChangeAnalysis(ctx, client, changeUuid, lf); err != nil {
		return err
	}

	// Call the simple RPC (enqueues a background job and returns immediately)
	_, err = client.StartChangeSimple(ctx, &connect.Request[sdp.StartChangeRequest]{
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

	waitForSnapshot := viper.GetBool("wait-for-snapshot")
	if waitForSnapshot {
		// Poll until change status has moved on
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
			status := changeResp.Msg.GetChange().GetMetadata().GetStatus()
			// Accept HAPPENING, or DONE: if an end-change was queued during
			// start-change, the worker kicks it off atomically and it may complete before
			// the next poll, advancing status to DONE. We must not poll indefinitely.
			if status == sdp.ChangeStatus_CHANGE_STATUS_HAPPENING ||
				status == sdp.ChangeStatus_CHANGE_STATUS_DONE {
				break
			}
			log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
				"status": status.String(),
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
		log.WithContext(ctx).WithFields(lf).Info("started change")
	} else {
		log.WithContext(ctx).WithFields(lf).Info("change start initiated (processing in background)")
	}
	return nil
}

func init() {
	changesCmd.AddCommand(startChangeCmd)

	addChangeUuidFlags(startChangeCmd)

	startChangeCmd.PersistentFlags().Bool("wait-for-snapshot", false, "Wait for the snapshot to complete before returning. Defaults to false.")
}
