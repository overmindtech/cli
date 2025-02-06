package cmd

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getAffectedBookmarksCmd represents the get-affected-bookmarks command
var getAffectedBookmarksCmd = &cobra.Command{
	Use:    "get-affected-bookmarks --snapshot-uuid ID --bookmark-uuids ID,ID,ID",
	Short:  "Calculates the bookmarks that would be overlapping with a snapshot.",
	PreRun: PreRunSetup,
	RunE:   GetAffectedBookmarks,
}

func GetAffectedBookmarks(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	snapshotUuid, err := uuid.Parse(viper.GetString("snapshot-uuid"))
	if err != nil {
		return flagError{usage: fmt.Sprintf("invalid --snapshot-uuid value '%v': %v\n\n%v", viper.GetString("snapshot-uuid"), err, cmd.UsageString())}
	}

	uuidStrings := viper.GetStringSlice("bookmark-uuids")
	bookmarkUuids := [][]byte{}
	for _, s := range uuidStrings {
		bookmarkUuid, err := uuid.Parse(s)
		if err != nil {
			return flagError{usage: fmt.Sprintf("invalid --bookmark-uuids value '%v': %v\n\n%v", bookmarkUuid, err, cmd.UsageString())}
		}
		bookmarkUuids = append(bookmarkUuids, bookmarkUuid[:])
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:read"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedBookmarkClient(ctx, oi)
	response, err := client.GetAffectedBookmarks(ctx, &connect.Request[sdp.GetAffectedBookmarksRequest]{
		Msg: &sdp.GetAffectedBookmarksRequest{
			SnapshotUUID:  snapshotUuid[:],
			BookmarkUUIDs: bookmarkUuids,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			message: "Failed to get affected bookmarks.",
		}
	}
	for _, u := range response.Msg.GetBookmarkUUIDs() {
		bookmarkUuid := uuid.UUID(u)
		log.WithContext(ctx).WithFields(log.Fields{
			"uuid": bookmarkUuid,
		}).Info("found affected bookmark")
	}
	return nil
}

func init() {
	bookmarksCmd.AddCommand(getAffectedBookmarksCmd)

	getAffectedBookmarksCmd.PersistentFlags().String("snapshot-uuid", "", "The UUID of the snapshot that should be checked.")
	getAffectedBookmarksCmd.PersistentFlags().String("bookmark-uuids", "", "A comma separated list of UUIDs of the potentially affected bookmarks.")
}
