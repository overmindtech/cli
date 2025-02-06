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

// getBookmarkCmd represents the get-bookmark command
var getBookmarkCmd = &cobra.Command{
	Use:    "get-bookmark --uuid ID",
	Short:  "Displays the contents of a bookmark.",
	PreRun: PreRunSetup,
	RunE:   GetBookmark,
}

func GetBookmark(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	bookmarkUuid, err := uuid.Parse(viper.GetString("uuid"))
	if err != nil {
		return flagError{
			usage: fmt.Sprintf("invalid --uuid value '%v' (%v)\n\n%v", viper.GetString("uuid"), err, cmd.UsageString()),
		}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:read"}, nil)
	if err != nil {
		return err
	}

	client := AuthenticatedBookmarkClient(ctx, oi)
	response, err := client.GetBookmark(ctx, &connect.Request[sdp.GetBookmarkRequest]{
		Msg: &sdp.GetBookmarkRequest{
			UUID: bookmarkUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to get bookmark",
		}
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"bookmark-uuid":        uuid.UUID(response.Msg.GetBookmark().GetMetadata().GetUUID()),
		"bookmark-created":     response.Msg.GetBookmark().GetMetadata().GetCreated().AsTime(),
		"bookmark-name":        response.Msg.GetBookmark().GetProperties().GetName(),
		"bookmark-description": response.Msg.GetBookmark().GetProperties().GetDescription(),
	}).Info("found bookmark")

	b, err := json.MarshalIndent(response.Msg.GetBookmark().ToMap(), "", "  ")
	if err != nil {
		log.Infof("Error rendering bookmark: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return nil
}

func init() {
	bookmarksCmd.AddCommand(getBookmarkCmd)

	getBookmarkCmd.PersistentFlags().String("uuid", "", "The UUID of the bookmark that should be displayed.")
}
