package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

// Commit ID, tag or branch name of the version of the assets that should be
// used in the comment. If the assets are updated, this should also be updated
// to reflect th latest version
//
// This allows us to update the assets without fear of breaking older comments
const assetVersion = "17c7fd2c365d4f4cdd8e414ca5148f825fa4febd"

func GetChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx, span := tracing.Tracer().Start(ctx, "CLI GetChange", trace.WithAttributes(
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

	lf := log.Fields{}
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]), true)
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	client := AuthenticatedChangesClient(ctx)
	var riskRes *connect.Response[sdp.GetChangeRisksResponse]
fetch:
	for {
		// use the variable to avoid shadowing
		var err error
		riskRes, err := client.GetChangeRisks(ctx, &connect.Request[sdp.GetChangeRisksRequest]{
			Msg: &sdp.GetChangeRisksRequest{
				UUID: changeUuid[:],
			},
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(log.Fields{
				"change-url": viper.GetString("change-url"),
			}).Error("failed to get change risks")
			return 1
		}

		if riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetStatus() == sdp.RiskCalculationStatus_STATUS_INPROGRESS {
			log.WithContext(ctx).WithField("status", riskRes.Msg.GetChangeRiskMetadata().GetRiskCalculationStatus().GetStatus().String()).Info("waiting for risk calculation")
			time.Sleep(10 * time.Second)
			// retry
		} else {
			// it's done (or errored)
			break fetch
		}
		if ctx.Err() != nil {
			log.WithContext(ctx).WithError(ctx.Err()).Error("context cancelled")
			return 1
		}
	}

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
		"change-uuid":        uuid.UUID(changeRes.Msg.GetChange().GetMetadata().GetUUID()),
		"change-created":     changeRes.Msg.GetChange().GetMetadata().GetCreatedAt().AsTime(),
		"change-status":      changeRes.Msg.GetChange().GetMetadata().GetStatus().String(),
		"change-name":        changeRes.Msg.GetChange().GetProperties().GetTitle(),
		"change-description": changeRes.Msg.GetChange().GetProperties().GetDescription(),
	}).Info("found change")

	switch viper.GetString("format") {
	case "json":
		b, err := json.MarshalIndent(changeRes.Msg.GetChange().ToMap(), "", "  ")
		if err != nil {
			log.WithContext(ctx).WithField("input", fmt.Sprintf("%#v", changeRes.Msg.GetChange().ToMap())).WithError(err).Error("Error rendering change")
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
			// Path to the assets folder on github
			AssetPath string
		}
		status := map[sdp.ItemDiffStatus]TemplateItem{
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED: {
				StatusAlt:  "unspecified",
				StatusIcon: "",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED: {
				StatusAlt:  "unchanged",
				StatusIcon: "item.svg",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED: {
				StatusAlt:  "created",
				StatusIcon: "created.svg",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED: {
				StatusAlt:  "updated",
				StatusIcon: "changed.svg",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED: {
				StatusAlt:  "deleted",
				StatusIcon: "deleted.svg",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED: {
				StatusAlt:  "replaced",
				StatusIcon: "replaced.svg",
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
				SeverityIcon: "low.svg",
				SeverityText: "Low",
			},
			sdp.Risk_SEVERITY_MEDIUM: {
				SeverityAlt:  "medium",
				SeverityIcon: "medium.svg",
				SeverityText: "Medium",
			},
			sdp.Risk_SEVERITY_HIGH: {
				SeverityAlt:  "high",
				SeverityIcon: "high.svg",
				SeverityText: "High",
			},
		}
		frontend, _ := strings.CutSuffix(viper.GetString("frontend"), "/")
		data := TemplateData{
			ChangeUrl:       fmt.Sprintf("%v/changes/%v", frontend, changeUuid.String()),
			ExpectedChanges: []TemplateItem{},
			UnmappedChanges: []TemplateItem{},
			BlastItems:      int(changeRes.Msg.GetChange().GetMetadata().GetNumAffectedItems()),
			BlastEdges:      int(changeRes.Msg.GetChange().GetMetadata().GetNumAffectedEdges()),
			Risks:           []TemplateRisk{},
			AssetPath:       fmt.Sprintf("https://raw.githubusercontent.com/overmindtech/ovm-cli/%v/assets", assetVersion),
		}

		for _, item := range changeRes.Msg.GetChange().GetProperties().GetPlannedChanges() {
			var before, after string
			if item.GetBefore() != nil {
				bb, err := yaml.Marshal(item.GetBefore().GetAttributes().GetAttrStruct().AsMap())
				if err != nil {
					log.WithContext(ctx).WithError(err).Error("error marshalling 'before' attributes")
					before = ""
				} else {
					before = string(bb)
				}
			}
			if item.GetAfter() != nil {
				ab, err := yaml.Marshal(item.GetAfter().GetAttributes().GetAttrStruct().AsMap())
				if err != nil {
					log.WithContext(ctx).WithError(err).Error("error marshalling 'after' attributes")
					after = ""
				} else {
					after = string(ab)
				}
			}
			edits := myers.ComputeEdits(diffspan.URIFromPath("current"), before, after)
			diff := fmt.Sprint(gotextdiff.ToUnified("current", "planned", before, edits))

			if item.GetItem() != nil {
				data.ExpectedChanges = append(data.ExpectedChanges, TemplateItem{
					StatusAlt:  status[item.GetStatus()].StatusAlt,
					StatusIcon: status[item.GetStatus()].StatusIcon,
					Type:       item.GetItem().GetType(),
					Title:      item.GetItem().GetUniqueAttributeValue(),
					Diff:       diff,
				})
			} else {
				var typ, title string
				if item.GetAfter() != nil {
					typ = item.GetAfter().GetType()
					title = item.GetAfter().UniqueAttributeValue()
				} else if item.GetBefore() != nil {
					typ = item.GetBefore().GetType()
					title = item.GetBefore().UniqueAttributeValue()
				}
				data.UnmappedChanges = append(data.UnmappedChanges, TemplateItem{
					StatusAlt:  status[item.GetStatus()].StatusAlt,
					StatusIcon: status[item.GetStatus()].StatusIcon,
					Type:       typ,
					Title:      title,
					Diff:       diff,
				})
			}
		}

		for _, risk := range riskRes.Msg.GetChangeRiskMetadata().GetRisks() {
			data.Risks = append(data.Risks, TemplateRisk{
				SeverityAlt:  severity[risk.GetSeverity()].SeverityAlt,
				SeverityIcon: severity[risk.GetSeverity()].SeverityIcon,
				SeverityText: severity[risk.GetSeverity()].SeverityText,
				Title:        risk.GetTitle(),
				Description:  risk.GetDescription(),
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
