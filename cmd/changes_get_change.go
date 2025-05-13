package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	diffspan "github.com/hexops/gotextdiff/span"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

//go:embed comment.md
var commentTemplate string

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:    "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short:  "Displays the contents of a change.",
	PreRun: PreRunSetup,
	RunE:   GetChange,
}

// Commit ID, tag or branch name of the version of the assets that should be
// used in the comment. If the assets are updated, this should also be updated
// to reflect the latest version
//
// This allows us to update the assets without fear of breaking older comments
const assetVersion = "476f6df5bc783c17b1d0513a43e0a0aa9c075588" // tag from v1.6.1

func GetChange(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	app := viper.GetString("app")

	riskLevels := []sdp.Risk_Severity{}
	for _, level := range viper.GetStringSlice("risk-levels") {
		switch level {
		case "high":
			riskLevels = append(riskLevels, sdp.Risk_SEVERITY_HIGH)
		case "medium":
			riskLevels = append(riskLevels, sdp.Risk_SEVERITY_MEDIUM)
		case "low":
			riskLevels = append(riskLevels, sdp.Risk_SEVERITY_LOW)
		default:
			return flagError{fmt.Sprintf("invalid --risk-levels value '%v', allowed values are 'high', 'medium', 'low'", level)}
		}
	}
	slices.Sort(riskLevels)
	riskLevels = slices.Compact(riskLevels)

	if len(riskLevels) == 0 {
		riskLevels = []sdp.Risk_Severity{sdp.Risk_SEVERITY_HIGH, sdp.Risk_SEVERITY_MEDIUM, sdp.Risk_SEVERITY_LOW}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:read"}, nil)
	if err != nil {
		return err
	}

	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]), viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
	}

	lf := log.Fields{
		"uuid":       changeUuid.String(),
		"change-url": viper.GetString("change-url"),
	}

	client := AuthenticatedChangesClient(ctx, oi)
	var timeLine *sdp.GetChangeTimelineV2Response
fetch:
	for {
		rawTimeLine, timelineErr := client.GetChangeTimelineV2(ctx, &connect.Request[sdp.GetChangeTimelineV2Request]{
			Msg: &sdp.GetChangeTimelineV2Request{
				ChangeUUID: changeUuid[:],
			},
		})
		if timelineErr != nil || rawTimeLine.Msg == nil {
			return loggedError{
				err:     timelineErr,
				fields:  lf,
				message: "failed to get timeline",
			}
		}
		timeLine = rawTimeLine.Msg
		for _, entry := range timeLine.GetEntries() {
			if entry.GetName() == string(sdp.ChangeTimelineEntryV2NameAutoTagging) && entry.GetStatus() == sdp.ChangeTimelineEntryStatus_DONE {
				break fetch
			}
		}
		// display the running entry
		runningEntry, status, err := sdp.TimelineFindInProgressEntry(timeLine.GetEntries())
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "failed to find running entry",
			}
		}
		// find the running timeline entry
		log.WithContext(ctx).WithFields(log.Fields{
			"status":  status.String(),
			"running": runningEntry,
		}).Info("Waiting for change analysis to complete")
		// retry
		time.Sleep(3 * time.Second)

		// check if the context is cancelled
		if ctx.Err() != nil {
			return loggedError{
				err:     ctx.Err(),
				fields:  lf,
				message: "context cancelled",
			}
		}
	}
	// get the change
	changeRes, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get change",
		}
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid":        uuid.UUID(changeRes.Msg.GetChange().GetMetadata().GetUUID()),
		"change-created":     changeRes.Msg.GetChange().GetMetadata().GetCreatedAt().AsTime(),
		"change-status":      changeRes.Msg.GetChange().GetMetadata().GetStatus().String(),
		"change-name":        changeRes.Msg.GetChange().GetProperties().GetTitle(),
		"change-description": changeRes.Msg.GetChange().GetProperties().GetDescription(),
	}).Info("found change")

	var calculateRiskStep *sdp.ChangeTimelineEntryV2
	for _, entry := range timeLine.GetEntries() {
		if entry.GetName() == string(sdp.ChangeTimelineEntryV2NameCalculatedRisks) {
			calculateRiskStep = entry
			break
		}
	}
	if calculateRiskStep == nil || calculateRiskStep.GetCalculatedRisks() == nil {
		return loggedError{
			err:     fmt.Errorf("failed to find risk calculation step"),
			fields:  lf,
			message: "failed to find risk calculation step",
		}
	}
	// filter the risks
	calculatedRisks := calculateRiskStep.GetCalculatedRisks().GetRisks()
	if len(riskLevels) != 3 {
		log.WithContext(ctx).WithFields(log.Fields{
			"risk-levels": renderRiskFilter(riskLevels),
		}).Info("filtering risks")

		calculatedRisks = filterRisks(calculatedRisks, riskLevels)
	}

	switch viper.GetString("format") {
	case "json":
		jsonStruct := struct {
			Change *sdp.Change `json:"change"`
			Risks  []*sdp.Risk `json:"risks"`
		}{
			Change: changeRes.Msg.GetChange(),
			Risks:  calculatedRisks,
		}

		b, err := json.MarshalIndent(jsonStruct, "", "  ")
		if err != nil {
			lf["input"] = fmt.Sprintf("%#v", jsonStruct)
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Error rendering change",
			}
		}

		fmt.Println(string(b))
	case "markdown":
		type TemplateItem struct {
			StatusSymbol string
			Type         string
			Title        string
			Diff         string
		}
		type TemplateRisk struct {
			SeverityText string
			Title        string
			Description  string
			RiskUrl      string
		}
		type TemplateData struct {
			BlastRadiusUrl  string
			ChangeUrl       string
			ExpectedChanges []TemplateItem
			UnmappedChanges []TemplateItem
			BlastItems      int
			BlastEdges      int
			Risks           []TemplateRisk
			// Path to the assets folder on github
			AssetPath string
			TagsLine  string
		}
		status := map[sdp.ItemDiffStatus]TemplateItem{
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED: {
				StatusSymbol: "-",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED: {
				StatusSymbol: "✓",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED: {
				StatusSymbol: "+",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED: {
				StatusSymbol: "~",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED: {
				StatusSymbol: "-",
			},
			sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED: {
				StatusSymbol: "+/-",
			},
		}

		severity := map[sdp.Risk_Severity]TemplateRisk{
			sdp.Risk_SEVERITY_UNSPECIFIED: {
				SeverityText: "unspecified",
			},
			sdp.Risk_SEVERITY_LOW: {
				SeverityText: "Low",
			},
			sdp.Risk_SEVERITY_MEDIUM: {
				SeverityText: "❗Medium",
			},
			sdp.Risk_SEVERITY_HIGH: {
				SeverityText: "‼️High",
			},
		}
		app, _ = strings.CutSuffix(app, "/")
		data := TemplateData{
			ChangeUrl:       fmt.Sprintf("%v/changes/%v", app, changeUuid.String()),
			BlastRadiusUrl:  fmt.Sprintf("%v/changes/%v/blast-radius", app, changeUuid.String()),
			ExpectedChanges: []TemplateItem{},
			UnmappedChanges: []TemplateItem{},
			BlastItems:      int(changeRes.Msg.GetChange().GetMetadata().GetNumAffectedItems()),
			BlastEdges:      int(changeRes.Msg.GetChange().GetMetadata().GetNumAffectedEdges()),
			Risks:           []TemplateRisk{},
			AssetPath:       fmt.Sprintf("https://raw.githubusercontent.com/overmindtech/cli/%v/assets", assetVersion),
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
					StatusSymbol: status[item.GetStatus()].StatusSymbol,
					Type:         item.GetItem().GetType(),
					Title:        item.GetItem().GetUniqueAttributeValue(),
					Diff:         diff,
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
					StatusSymbol: status[item.GetStatus()].StatusSymbol,
					Type:         typ,
					Title:        title,
					Diff:         diff,
				})
			}
		}

		for _, risk := range calculatedRisks {
			// parse the risk UUID to a string
			riskUuid, _ := uuid.FromBytes(risk.GetUUID())
			data.Risks = append(data.Risks, TemplateRisk{
				SeverityText: severity[risk.GetSeverity()].SeverityText,
				Title:        risk.GetTitle(),
				Description:  risk.GetDescription(),
				RiskUrl:      fmt.Sprintf("%v/changes/%v/blast-radius?selectedRisk=%v&activeTab=risks", app, changeUuid.String(), riskUuid.String()),
			})
		}
		// get the tags in
		data.TagsLine = getTagsLine(changeRes.Msg.GetChange().GetProperties().GetEnrichedTags().GetTagValue())
		data.TagsLine = strings.TrimSpace(data.TagsLine)

		tmpl, err := template.New("comment").Parse(commentTemplate)
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "error parsing comment template",
			}
		}
		err = tmpl.Execute(os.Stdout, data)
		if err != nil {
			lf["input"] = fmt.Sprintf("%#v", data)
			return loggedError{
				err:     err,
				fields:  lf,
				message: "error rendering comment",
			}
		}
	}

	return nil
}

func filterRisks(risks []*sdp.Risk, levels []sdp.Risk_Severity) []*sdp.Risk {
	filteredRisks := make([]*sdp.Risk, 0)

	for _, risk := range risks {
		if slices.Contains(levels, risk.GetSeverity()) {
			filteredRisks = append(filteredRisks, risk)
		}
	}

	return filteredRisks
}

func renderRiskFilter(levels []sdp.Risk_Severity) string {
	result := make([]string, 0, len(levels))
	for _, level := range levels {
		switch level {
		case sdp.Risk_SEVERITY_HIGH:
			result = append(result, "high")
		case sdp.Risk_SEVERITY_MEDIUM:
			result = append(result, "medium")
		case sdp.Risk_SEVERITY_LOW:
			result = append(result, "low")
		case sdp.Risk_SEVERITY_UNSPECIFIED:
			continue
		}
	}
	return strings.Join(result, ", ")
}

func getTagsLine(tags map[string]*sdp.TagValue) string {
	autoTags := ""
	userTags := ""

	for key, value := range tags {
		if value.GetAutoTagValue() != nil {
			suffix := ""
			if value.GetAutoTagValue().GetValue() != "" {
				suffix = fmt.Sprintf("|%s", value.GetAutoTagValue().GetValue())
			}
			autoTags += fmt.Sprintf("`✨%s%s` ", key, suffix)
		} else if value.GetUserTagValue() != nil {
			suffix := ""
			if value.GetUserTagValue().GetValue() != "" {
				suffix = fmt.Sprintf("|%s", value.GetUserTagValue().GetValue())
			}
			userTags += fmt.Sprintf("`%s%s` ", key, suffix)
		} else {
			// we should never get here, but just in case.
			// its a tag jim, but not as we know it
			userTags += fmt.Sprintf("`%s` ", key)
		}
	}
	return autoTags + userTags
}

func init() {
	changesCmd.AddCommand(getChangeCmd)
	addAPIFlags(getChangeCmd)

	addChangeUuidFlags(getChangeCmd)
	getChangeCmd.PersistentFlags().String("status", "CHANGE_STATUS_DEFINING", "The expected status of the change. Use this with --ticket-link to get the first change with that status for a given ticket link. Allowed values: CHANGE_STATUS_UNSPECIFIED, CHANGE_STATUS_DEFINING, CHANGE_STATUS_HAPPENING, CHANGE_STATUS_PROCESSING, CHANGE_STATUS_DONE")

	getChangeCmd.PersistentFlags().String("frontend", "", "The frontend base URL")
	_ = submitPlanCmd.PersistentFlags().MarkDeprecated("frontend", "This flag is no longer used and will be removed in a future release. Use the '--app' flag instead.") // MarkDeprecated only errors if the flag doesn't exist, we fall back to using app
	getChangeCmd.PersistentFlags().String("format", "json", "How to render the change. Possible values: json, markdown")
	getChangeCmd.PersistentFlags().StringSlice("risk-levels", []string{"high", "medium", "low"}, "Only show changes with the specified risk levels. Allowed values: high, medium, low")
}
