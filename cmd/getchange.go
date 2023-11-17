package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	diffspan "github.com/hexops/gotextdiff/span"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

//go:embed comment.md
var commentTemplate string

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:   "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
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

		exitcode := GetChange(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI GetChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
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

	lf := log.Fields{}
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]), true)
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	client := AuthenticatedChangesClient(ctx)
	changeRes, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"change-url": viper.GetString("change-url"),
		}).Error("failed to get change")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid":        uuid.UUID(changeRes.Msg.Change.Metadata.UUID),
		"change-created":     changeRes.Msg.Change.Metadata.CreatedAt.AsTime(),
		"change-status":      changeRes.Msg.Change.Metadata.Status.String(),
		"change-name":        changeRes.Msg.Change.Properties.Title,
		"change-description": changeRes.Msg.Change.Properties.Description,
	}).Info("found change")

	// diffRes, err := client.GetDiff(ctx, &connect.Request[sdp.GetDiffRequest]{
	// 	Msg: &sdp.GetDiffRequest{
	// 		ChangeUUID: changeUuid[:],
	// 	},
	// })
	// if err != nil {
	// 	log.WithContext(ctx).WithError(err).WithFields(log.Fields{
	// 		"change-url": viper.GetString("change-url"),
	// 	}).Error("failed to get change diff")
	// 	return 1
	// }
	// log.WithContext(ctx).WithFields(log.Fields{
	// 	"change-uuid": uuid.UUID(changeRes.Msg.Change.Metadata.UUID),
	// }).Info("loaded change diff")

	switch viper.GetString("format") {
	case "json":
		b, err := json.MarshalIndent(changeRes.Msg.Change.ToMap(), "", "  ")
		if err != nil {
			log.WithContext(ctx).WithField("input", fmt.Sprintf("%#v", changeRes.Msg.Change.ToMap())).WithError(err).Error("Error rendering change")
			return 1
		}

		fmt.Println(string(b))
	case "markdown":
		type TemplateItem struct {
			StatusAlt  string
			StatusIcon string
			Type       string
			Title      string
			Diff       string
		}
		type TemplateRisk struct {
			SeverityAlt  string
			SeverityIcon string
			SeverityText string
			Title        string
			Description  string
		}
		type TemplateData struct {
			ChangeUrl       string
			ExpectedChanges []TemplateItem
			UnmappedChanges []TemplateItem
			BlastItems      int
			BlastEdges      int
			Risks           []TemplateRisk
		}
		status := map[sdp.ItemDiffStatus]TemplateItem{
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED: {
				StatusAlt:  "unspecified",
				StatusIcon: "",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED: {
				StatusAlt:  "unchanged",
				StatusIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/item.png",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED: {
				StatusAlt:  "created",
				StatusIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/created.png",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED: {
				StatusAlt:  "updated",
				StatusIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/changed.png",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED: {
				StatusAlt:  "deleted",
				StatusIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/deleted.png",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED: {
				StatusAlt:  "replaced",
				StatusIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/replaced.png",
			},
		}

		severity := map[sdp.Risk_Severity]TemplateRisk{
			sdp.Risk_SEVERITY_UNSPECIFIED: {
				SeverityAlt:  "unspecified",
				SeverityIcon: "",
				SeverityText: "unspecified",
			},
			sdp.Risk_SEVERITY_LOW: {
				SeverityAlt:  "low",
				SeverityIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/low.png",
				SeverityText: "Low",
			},
			sdp.Risk_SEVERITY_MEDIUM: {
				SeverityAlt:  "medium",
				SeverityIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/medium.png",
				SeverityText: "Medium",
			},
			sdp.Risk_SEVERITY_HIGH: {
				SeverityAlt:  "high",
				SeverityIcon: "https://raw.githubusercontent.com/overmindtech/ovm-cli/ac4feb1b9dd73b5c42c5a515d12517b551d2886b/assets/high.png",
				SeverityText: "High",
			},
		}
		data := TemplateData{
			ChangeUrl:       fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), changeUuid.String()),
			ExpectedChanges: []TemplateItem{},
			UnmappedChanges: []TemplateItem{},
			BlastItems:      75,
			BlastEdges:      97,
			Risks:           []TemplateRisk{},
		}

		for _, item := range changeRes.Msg.Change.Properties.PlannedChanges {
			var before, after string
			if item.Before != nil {
				bb, err := yaml.Marshal(item.Before.Attributes.AttrStruct.AsMap())
				if err != nil {
					log.WithContext(ctx).WithError(err).Error("error marshalling 'before' attributes")
					before = ""
				} else {
					before = string(bb)
				}
			}
			if item.After != nil {
				ab, err := yaml.Marshal(item.After.Attributes.AttrStruct.AsMap())
				if err != nil {
					log.WithContext(ctx).WithError(err).Error("error marshalling 'after' attributes")
					after = ""
				} else {
					after = string(ab)
				}
			}
			edits := myers.ComputeEdits(diffspan.URIFromPath("current"), before, after)
			diff := fmt.Sprint(gotextdiff.ToUnified("current", "planned", before, edits))

			if item.Item != nil {
				data.ExpectedChanges = append(data.ExpectedChanges, TemplateItem{
					StatusAlt:  status[item.Status].StatusAlt,
					StatusIcon: status[item.Status].StatusIcon,
					Type:       item.Item.Type,
					Title:      item.Item.UniqueAttributeValue,
					Diff:       diff,
				})
			} else {
				var typ, title string
				if item.After != nil {
					typ = item.After.Type
					title = item.After.UniqueAttributeValue()
				} else if item.Before != nil {
					typ = item.Before.Type
					title = item.Before.UniqueAttributeValue()
				}
				data.UnmappedChanges = append(data.UnmappedChanges, TemplateItem{
					StatusAlt:  status[item.Status].StatusAlt,
					StatusIcon: status[item.Status].StatusIcon,
					Type:       typ,
					Title:      title,
					Diff:       diff,
				})
			}
		}

		for _, risk := range changeRes.Msg.Change.Metadata.Risks {
			data.Risks = append(data.Risks, TemplateRisk{
				SeverityAlt:  severity[risk.Severity].SeverityAlt,
				SeverityIcon: severity[risk.Severity].SeverityIcon,
				SeverityText: severity[risk.Severity].SeverityText,
				Title:        risk.Title,
				Description:  risk.Description,
			})
		}

		tmpl, err := template.New("comment").Parse(commentTemplate)
		if err != nil {
			log.WithContext(ctx).WithError(err).Error("error parsing comment template")
			return 1
		}
		err = tmpl.Execute(os.Stdout, data)
		if err != nil {
			log.WithContext(ctx).WithField("input", fmt.Sprintf("%#v", data)).WithError(err).Error("error rendering comment")
			return 1
		}
	}

	return 0
}

func init() {
	rootCmd.AddCommand(getChangeCmd)

	withChangeUuidFlags(getChangeCmd)
	getChangeCmd.PersistentFlags().String("status", "", "The expected status of the change. Use this with --ticket-link. Allowed values: CHANGE_STATUS_UNSPECIFIED, CHANGE_STATUS_DEFINING, CHANGE_STATUS_HAPPENING, CHANGE_STATUS_PROCESSING, CHANGE_STATUS_DONE")

	getChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")
	getChangeCmd.PersistentFlags().String("format", "json", "How to render the change. Possible values: json, markdown")

	getChangeCmd.PersistentFlags().String("timeout", "5m", "How long to wait for responses")
}
