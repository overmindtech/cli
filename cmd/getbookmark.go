package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goroutine to watch for cancellation signals
		go func() {
			select {
			case <-sigs:
				cancel()
			case <-ctx.Done():
			}
		}()

		exitcode := GetBookmark(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetBookmark(ctx context.Context, ready chan bool) int {
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

	ctx, span := tracing.Tracer().Start(ctx, "CLI GetBookmark", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, []string{"changes:read"})
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

	return 0
}

func init() {
	rootCmd.AddCommand(getBookmarkCmd)

	getBookmarkCmd.PersistentFlags().String("bookmark-url", "", "The bookmark service API endpoint (defaults to --url)")

	getBookmarkCmd.PersistentFlags().String("uuid", "", "The UUID of the bookmark that should be displayed.")

	getBookmarkCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
