package cmd

import (
	"context"
	"encoding/json"
	"errors"
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

// listChangesCmd represents the get-change command
var listChangesCmd = &cobra.Command{
	Use:   "list-changes --dir ./output",
	Short: "Displays the contents of a change.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-change` flags")
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

		exitcode := ListChanges(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func ListChanges(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI ListChanges", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, []string{"changes:read"})
	if err != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	snapshots := AuthenticatedSnapshotsClient(ctx)
	bookmarks := AuthenticatedBookmarkClient(ctx)
	changes := AuthenticatedChangesClient(ctx)

	response, err := changes.ListChanges(ctx, &connect.Request[sdp.ListChangesRequest]{
		Msg: &sdp.ListChangesRequest{},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("failed to list changes")
		return 1
	}
	for _, change := range response.Msg.GetChanges() {
		changeUuid := uuid.UUID(change.GetMetadata().GetUUID())
		log.WithContext(ctx).WithFields(log.Fields{
			"change-uuid":        changeUuid,
			"change-created":     change.GetMetadata().GetCreatedAt().AsTime(),
			"change-status":      change.GetMetadata().GetStatus().String(),
			"change-name":        change.GetProperties().GetTitle(),
			"change-description": change.GetProperties().GetDescription(),
		}).Info("found change")

		b, err := json.MarshalIndent(change.ToMap(), "", "  ")
		if err != nil {
			log.WithContext(ctx).Errorf("Error rendering change: %v", err)
			return 1
		}

		err = printJson(ctx, b, "change", changeUuid.String())
		if err != nil {
			return 1
		}

		if viper.GetBool("fetch-data") {
			ciUuid := uuid.UUID(change.GetProperties().GetChangingItemsBookmarkUUID())
			if ciUuid != uuid.Nil {
				changingItems, err := bookmarks.GetBookmark(ctx, &connect.Request[sdp.GetBookmarkRequest]{
					Msg: &sdp.GetBookmarkRequest{
						UUID: ciUuid[:],
					},
				})
				// continue processing if item not found
				if connect.CodeOf(err) != connect.CodeNotFound {
					if err != nil {
						log.WithContext(ctx).WithError(err).WithFields(log.Fields{
							"change-uuid":         changeUuid,
							"changing-items-uuid": ciUuid.String(),
						}).Error("failed to get ChangingItemsBookmark")
						return 1
					}

					b, err := json.MarshalIndent(changingItems.Msg.GetBookmark().ToMap(), "", "  ")
					if err != nil {
						log.WithContext(ctx).WithFields(log.Fields{
							"change-uuid":         changeUuid,
							"changing-items-uuid": ciUuid.String(),
						}).Errorf("Error rendering changing items bookmark: %v", err)
						return 1
					}

					err = printJson(ctx, b, "changing-items", ciUuid.String())
					if err != nil {
						return 1
					}
				}
			}

			brUuid := uuid.UUID(change.GetProperties().GetBlastRadiusSnapshotUUID())
			if brUuid != uuid.Nil {
				brSnap, err := snapshots.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
					Msg: &sdp.GetSnapshotRequest{
						UUID: brUuid[:],
					},
				})
				// continue processing if item not found
				if connect.CodeOf(err) != connect.CodeNotFound {
					if err != nil {
						log.WithContext(ctx).WithError(err).WithFields(log.Fields{
							"change-uuid":       changeUuid,
							"blast-radius-uuid": brUuid.String(),
						}).Error("failed to get BlastRadiusSnapshot")
						return 1
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						log.WithContext(ctx).WithFields(log.Fields{
							"change-uuid":       changeUuid,
							"blast-radius-uuid": brUuid.String(),
						}).Errorf("Error rendering blast radius snapshot: %v", err)
						return 1
					}

					err = printJson(ctx, b, "blast-radius", brUuid.String())
					if err != nil {
						return 1
					}
				}
			}

			sbsUuid := uuid.UUID(change.GetProperties().GetSystemBeforeSnapshotUUID())
			if sbsUuid != uuid.Nil {
				brSnap, err := snapshots.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
					Msg: &sdp.GetSnapshotRequest{
						UUID: sbsUuid[:],
					},
				})
				// continue processing if item not found
				if connect.CodeOf(err) != connect.CodeNotFound {
					if err != nil {
						log.WithContext(ctx).WithError(err).WithFields(log.Fields{
							"change-uuid":        changeUuid,
							"system-before-uuid": sbsUuid.String(),
						}).Error("failed to get SystemBeforeSnapshot")
						return 1
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						log.WithContext(ctx).WithFields(log.Fields{
							"change-uuid":        changeUuid,
							"system-before-uuid": sbsUuid.String(),
						}).Errorf("Error rendering system before snapshot: %v", err)
						return 1
					}

					err = printJson(ctx, b, "system-before", sbsUuid.String())
					if err != nil {
						return 1
					}
				}
			}

			sasUuid := uuid.UUID(change.GetProperties().GetSystemAfterSnapshotUUID())
			if sasUuid != uuid.Nil {
				brSnap, err := snapshots.GetSnapshot(ctx, &connect.Request[sdp.GetSnapshotRequest]{
					Msg: &sdp.GetSnapshotRequest{
						UUID: sasUuid[:],
					},
				})
				// continue processing if item not found
				if connect.CodeOf(err) != connect.CodeNotFound {
					if err != nil {
						log.WithContext(ctx).WithError(err).WithFields(log.Fields{
							"change-uuid":       changeUuid,
							"system-after-uuid": sasUuid.String(),
						}).Error("failed to get SystemAfterSnapshot")
						return 1
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						log.WithContext(ctx).WithFields(log.Fields{
							"change-uuid":       changeUuid,
							"system-after-uuid": sasUuid.String(),
						}).Errorf("Error rendering system after snapshot: %v", err)
						return 1
					}

					err = printJson(ctx, b, "system-after", sasUuid.String())
					if err != nil {
						return 1
					}
				}
			}
		}
	}

	return 0
}

func printJson(ctx context.Context, b []byte, prefix, id string) error {
	switch viper.GetString("format") {
	case "json":
		fmt.Println(string(b))
	case "files":
		dir := viper.GetString("dir")
		if dir == "" {
			return errors.New("need --dir value to write to files")
		}

		// write the change to a file
		fileName := fmt.Sprintf("%v/%v-%v.json", dir, prefix, id)
		file, err := os.Create(fileName)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"prefix":      prefix,
				"id":          id,
				"output-dir":  dir,
				"output-file": fileName,
			}).Error("failed to create file")
			return err
		}

		_, err = file.Write(b)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"prefix":      prefix,
				"id":          id,
				"output-dir":  dir,
				"output-file": fileName,
			}).Error("failed to write file")
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(listChangesCmd)

	listChangesCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")
	listChangesCmd.PersistentFlags().String("format", "files", "How to render the change. Possible values: files, json")
	listChangesCmd.PersistentFlags().String("dir", "./output", "A directory name to use for rendering changes when using the 'files' format")
	listChangesCmd.PersistentFlags().Bool("fetch-data", false, "also fetch the blast radius and system state snapshots for each change")

	listChangesCmd.PersistentFlags().String("timeout", "5m", "How long to wait for responses")
}
