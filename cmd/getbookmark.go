package cmd

import (
	"context"
	"fmt"
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

// getBookmarkCmd represents the get-bookmark command
var getBookmarkCmd = &cobra.Command{
	Use:   "get-bookmark --uuid ID",
	Short: "Displays the contents of a bookmark.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-bookmark` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := GetBookmark(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetBookmark(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	bookmarkUuid, err := uuid.Parse(viper.GetString("uuid"))
	if err != nil {
		log.Errorf("invalid --uuid value '%v', error: %v", viper.GetString("uuid"), err)
		return 1
	}

	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI GetBookmark", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedBookmarkClient(ctx)
	response, err := client.GetBookmark(ctx, &connect.Request[sdp.GetBookmarkRequest]{
		Msg: &sdp.GetBookmarkRequest{
			UUID: bookmarkUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"bookmark-url": viper.GetString("bookmark-url"),
		}).Error("failed to get bookmark")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"bookmark-uuid":        response.Msg.Bookmark.Metadata.UUID,
		"bookmark-created":     response.Msg.Bookmark.Metadata.Created,
		"bookmark-name":        response.Msg.Bookmark.Properties.Name,
		"bookmark-description": response.Msg.Bookmark.Properties.Description,
	}).Info("found bookmark")
	for _, q := range response.Msg.Bookmark.Properties.Queries {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-query": q,
		}).Info("found bookmark query")
	}
	for _, i := range response.Msg.Bookmark.Properties.ExcludedItems {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-excluded-item": i,
		}).Info("found bookmark excluded item")
	}
	return 0
}

func init() {
	rootCmd.AddCommand(getBookmarkCmd)

	getBookmarkCmd.PersistentFlags().String("bookmark-url", "", "The bookmark service API endpoint (defaults to --url)")
	getBookmarkCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")

	getBookmarkCmd.PersistentFlags().String("uuid", "", "The UUID of the bookmark that should be displayed.")

	getBookmarkCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
