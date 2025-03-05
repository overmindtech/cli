package cmd

import (
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getSnapshotCmd represents the get-snapshot command
var getSnapshotCmd = &cobra.Command{
	Use:    "get-snapshot --uuid ID",
	Short:  "Displays the contents of a snapshot.",
	PreRun: PreRunSetup,
	RunE:   GetSnapshot,
}

func GetSnapshot(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	snapshotUuid, err := uuid.Parse(viper.GetString("uuid"))
	if err != nil {
		return flagError{usage: fmt.Sprintf("invalid --uuid value '%v', error: %v\n\n%v", viper.GetString("uuid"), err, cmd.UsageString())}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"explore:read", "changes:read"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedSnapshotsClient(ctx, oi)
	response, err := client.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
		Msg: &sdp.GetSnapshotRequest{
			UUID: snapshotUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to get snapshot",
		}
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

	return nil
}

func init() {
	snapshotsCmd.AddCommand(getSnapshotCmd)

	getSnapshotCmd.PersistentFlags().String("uuid", "", "The UUID of the snapshot that should be displayed.")
}
