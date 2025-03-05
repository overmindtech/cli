package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createBookmarkCmd represents the get-bookmark command
var createBookmarkCmd = &cobra.Command{
	Use:    "create-bookmark [--file FILE]",
	Short:  "Creates a bookmark from JSON.",
	PreRun: PreRunSetup,
	RunE:   CreateBookmark,
}

func CreateBookmark(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var err error

	in := os.Stdin
	if viper.GetString("file") != "" {
		in, err = os.Open(viper.GetString("file"))
		if err != nil {
			return loggedError{
				err: err,
				fields: log.Fields{
					"file": viper.GetString("file"),
				},
				message: "failed to open input",
			}
		}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"}, nil)
	if err != nil {
		return err
	}

	contents, err := io.ReadAll(in)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  log.Fields{"file": viper.GetString("file")},
			message: "failed to read file",
		}
	}
	msg := sdp.BookmarkProperties{}
	err = json.Unmarshal(contents, &msg)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to parse input",
		}
	}
	client := AuthenticatedBookmarkClient(ctx, oi)
	response, err := client.CreateBookmark(ctx, &connect.Request[sdp.CreateBookmarkRequest]{
		Msg: &sdp.CreateBookmarkRequest{
			Properties: &msg,
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
		"bookmark-created":     response.Msg.GetBookmark().GetMetadata().GetCreated(),
		"bookmark-name":        response.Msg.GetBookmark().GetProperties().GetName(),
		"bookmark-description": response.Msg.GetBookmark().GetProperties().GetDescription(),
	}).Info("created bookmark")
	for _, q := range response.Msg.GetBookmark().GetProperties().GetQueries() {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-query": q,
		}).Info("created bookmark query")
	}

	b, err := json.MarshalIndent(response.Msg.GetBookmark().GetProperties(), "", "  ")
	if err != nil {
		log.Infof("Error rendering bookmark: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return nil
}

func init() {
	bookmarksCmd.AddCommand(createBookmarkCmd)

	createBookmarkCmd.PersistentFlags().String("file", "", "JSON formatted file to read bookmark. (defaults to stdin)")
}
