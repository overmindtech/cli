package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// listChangesCmd represents the get-change command
var listChangesCmd = &cobra.Command{
	Use:    "list-changes --dir ./output",
	Short:  "Displays the contents of a change.",
	PreRun: PreRunSetup,
	RunE:   ListChanges,
}

func ListChanges(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:read"}, nil)
	if err != nil {
		return err
	}

	snapshots := AuthenticatedSnapshotsClient(ctx, oi)
	bookmarks := AuthenticatedBookmarkClient(ctx, oi)
	changes := AuthenticatedChangesClient(ctx, oi)

	response, err := changes.ListChanges(ctx, &connect.Request[sdp.ListChangesRequest]{
		Msg: &sdp.ListChangesRequest{},
	})
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to list changes",
		}
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
			return loggedError{
				err:     err,
				message: "Error rendering change",
			}
		}

		err = printJson(ctx, b, "change", changeUuid.String(), cmd)
		if err != nil {
			return err
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
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":         changeUuid,
								"changing-items-uuid": ciUuid.String(),
							},
							message: "failed to get ChangingItemsBookmark",
						}
					}

					b, err := json.MarshalIndent(changingItems.Msg.GetBookmark().ToMap(), "", "  ")
					if err != nil {
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":         changeUuid,
								"changing-items-uuid": ciUuid.String(),
							},
							message: "Error rendering changing items bookmark",
						}
					}

					err = printJson(ctx, b, "changing-items", ciUuid.String(), cmd)
					if err != nil {
						return err
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
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":       changeUuid,
								"blast-radius-uuid": brUuid.String(),
							},
							message: "failed to get BlastRadiusSnapshot",
						}
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":       changeUuid,
								"blast-radius-uuid": brUuid.String(),
							},
							message: "Error rendering blast radius snapshot",
						}
					}

					err = printJson(ctx, b, "blast-radius", brUuid.String(), cmd)
					if err != nil {
						return err
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
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":        changeUuid,
								"system-before-uuid": sbsUuid.String(),
							},
							message: "failed to get SystemBeforeSnapshot",
						}
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":        changeUuid,
								"system-before-uuid": sbsUuid.String(),
							},
							message: "Error rendering system before snapshot",
						}
					}

					err = printJson(ctx, b, "system-before", sbsUuid.String(), cmd)
					if err != nil {
						return err
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
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":       changeUuid,
								"system-after-uuid": sasUuid.String(),
							},
							message: "failed to get SystemAfterSnapshot",
						}
					}

					b, err := json.MarshalIndent(brSnap.Msg.GetSnapshot().ToMap(), "", "  ")
					if err != nil {
						return loggedError{
							err: err,
							fields: log.Fields{
								"change-uuid":       changeUuid,
								"system-after-uuid": sasUuid.String(),
							},
							message: "Error rendering system after snapshot",
						}
					}

					err = printJson(ctx, b, "system-after", sasUuid.String(), cmd)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func printJson(_ context.Context, b []byte, prefix, id string, cmd *cobra.Command) error {
	switch viper.GetString("format") {
	case "json":
		fmt.Println(string(b))
	case "files":
		dir := viper.GetString("dir")
		if dir == "" {
			return flagError{fmt.Sprintf("need --dir value to write to files\n\n%v", cmd.UsageString())}
		}

		// attempt to create the directory
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return loggedError{
				err: err,
				fields: log.Fields{
					"output-dir": dir,
				},
				message: "failed to create output directory",
			}
		}

		// write the change to a file
		fileName := fmt.Sprintf("%v/%v-%v.json", dir, prefix, id)
		file, err := os.Create(fileName)
		if err != nil {
			return loggedError{
				err: err,
				fields: log.Fields{
					"prefix":      prefix,
					"id":          id,
					"output-dir":  dir,
					"output-file": fileName,
				},
				message: "failed to create file",
			}
		}

		_, err = file.Write(b)
		if err != nil {
			return loggedError{
				err: err,
				fields: log.Fields{
					"prefix":      prefix,
					"id":          id,
					"output-dir":  dir,
					"output-file": fileName,
				},
				message: "failed to write file",
			}
		}
	}

	return nil
}

func init() {
	changesCmd.AddCommand(listChangesCmd)

	listChangesCmd.PersistentFlags().String("format", "files", "How to render the change. Possible values: files, json")
	listChangesCmd.PersistentFlags().String("dir", "./output", "A directory name to use for rendering changes when using the 'files' format")
	listChangesCmd.PersistentFlags().Bool("fetch-data", false, "also fetch the blast radius and system state snapshots for each change")
}
