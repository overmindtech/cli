package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// createBookmarkCmd represents the get-bookmark command
var createBookmarkCmd = &cobra.Command{
	Use:   "create-bookmark [--file FILE]",
	Short: "Creates a bookmark from JSON.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `create-bookmark` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := CreateBookmark(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func CreateBookmark(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	in := os.Stdin
	if viper.GetString("file") != "" {
		in, err = os.Open(viper.GetString("file"))
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"file": viper.GetString("file"),
			}).Error("failed to open input")
			return 1
		}
	}

	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI CreateBookmark", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, []string{"changes:write"}, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	contents, err := io.ReadAll(in)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to read file")
		return 1
	}
	msg := sdp.BookmarkProperties{}
	err = json.Unmarshal(contents, &msg)
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to parse input")
		return 1
	}
	client := AuthenticatedBookmarkClient(ctx)
	response, err := client.CreateBookmark(ctx, &connect.Request[sdp.CreateBookmarkRequest]{
		Msg: &sdp.CreateBookmarkRequest{
			Properties: &msg,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"bookmark-url": viper.GetString("bookmark-url"),
		}).Error("failed to get bookmark")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"bookmark-uuid":        uuid.UUID(response.Msg.Bookmark.Metadata.UUID),
		"bookmark-created":     response.Msg.Bookmark.Metadata.Created,
		"bookmark-name":        response.Msg.Bookmark.Properties.Name,
		"bookmark-description": response.Msg.Bookmark.Properties.Description,
	}).Info("created bookmark")
	for _, q := range response.Msg.Bookmark.Properties.Queries {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-query": q,
		}).Info("created bookmark query")
	}
	for _, i := range response.Msg.Bookmark.Properties.ExcludedItems {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-excluded-item": i,
		}).Info("created bookmark excluded item")
	}

	b, err := json.MarshalIndent(response.Msg.Bookmark.Properties, "", "  ")
	if err != nil {
		log.Infof("Error rendering bookmark: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return 0
}

func init() {
	rootCmd.AddCommand(createBookmarkCmd)

	createBookmarkCmd.PersistentFlags().String("bookmark-url", "", "The bookmark service API endpoint (defaults to --url)")

	createBookmarkCmd.PersistentFlags().String("file", "", "JSON formatted file to read bookmark. (defaults to stdin)")

	createBookmarkCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
