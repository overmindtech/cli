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

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
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

		exitcode := CreateBookmark(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func CreateBookmark(ctx context.Context, ready chan bool) int {
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

	ctx, span := tracing.Tracer().Start(ctx, "CLI CreateBookmark", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", tracedSettings())),
	))
	defer span.End()

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get instance data from app")
		return 1
	}

	ctx, _, err = ensureToken(ctx, oi, []string{"changes:write"})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to authenticate")
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
	client := AuthenticatedBookmarkClient(ctx, oi)
	response, err := client.CreateBookmark(ctx, &connect.Request[sdp.CreateBookmarkRequest]{
		Msg: &sdp.CreateBookmarkRequest{
			Properties: &msg,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get bookmark")
		return 1
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
	for _, i := range response.Msg.GetBookmark().GetProperties().GetExcludedItems() {
		log.WithContext(ctx).WithFields(log.Fields{
			"bookmark-excluded-item": i,
		}).Info("created bookmark excluded item")
	}

	b, err := json.MarshalIndent(response.Msg.GetBookmark().GetProperties(), "", "  ")
	if err != nil {
		log.Infof("Error rendering bookmark: %v", err)
	} else {
		fmt.Println(string(b))
	}

	return 0
}

func init() {
	bookmarksCmd.AddCommand(createBookmarkCmd)

	createBookmarkCmd.PersistentFlags().String("file", "", "JSON formatted file to read bookmark. (defaults to stdin)")
}
