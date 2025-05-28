package cmd

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:    "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short:  "Displays the contents of a change.",
	PreRun: PreRunSetup,
	RunE:   GetChange,
}

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
	app, _ = strings.CutSuffix(app, "/")
	// get the change
	var format sdp.ChangeOutputFormat
	switch viper.GetString("format") {
	case "json":
		format = sdp.ChangeOutputFormat_CHANGE_OUTPUT_FORMAT_JSON
	case "markdown":
		format = sdp.ChangeOutputFormat_CHANGE_OUTPUT_FORMAT_MARKDOWN
	default:
		return fmt.Errorf("Unknown output format. Please select 'json' or 'markdown'")
	}
	changeRes, err := client.GetChangeSummary(ctx, &connect.Request[sdp.GetChangeSummaryRequest]{
		Msg: &sdp.GetChangeSummaryRequest{
			UUID:               changeUuid[:],
			ChangeOutputFormat: format,
			RiskSeverityFilter: riskLevels,
			AppURL:             app,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get change summary",
		}
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid": changeUuid.String(),
	}).Info("found change")

	fmt.Println(changeRes.Msg.GetChange())

	return nil
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
