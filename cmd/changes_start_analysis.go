package cmd

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startAnalysisCmd represents the start-analysis command
var startAnalysisCmd = &cobra.Command{
	Use:    "start-analysis {--ticket-link URL | --uuid ID | --change URL}",
	Short:  "Triggers analysis on a change with previously stored planned changes",
	Long: `Triggers analysis on a change that has previously stored planned changes.

This command is used in multi-plan workflows (e.g., Atlantis parallel planning) where
multiple terraform plans are submitted independently using 'submit-plan --no-start',
and then analysis is triggered once all plans are submitted.

The change must be in DEFINING status and must have at least one planned change stored.`,
	PreRun: PreRunSetup,
	RunE:   StartAnalysis,
}

func StartAnalysis(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	app := viper.GetString("app")

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write", "sources:read"}, nil)
	if err != nil {
		return err
	}

	lf := log.Fields{}

	changeUUID, err := getChangeUUIDAndCheckStatus(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to identify change",
		}
	}

	lf["change"] = changeUUID.String()

	analysisConfig, err := buildAnalysisConfig(ctx, lf)
	if err != nil {
		return err
	}

	client := AuthenticatedChangesClient(ctx, oi)

	_, err = client.StartChangeAnalysis(ctx, &connect.Request[sdp.StartChangeAnalysisRequest]{
		Msg: &sdp.StartChangeAnalysisRequest{
			ChangeUUID:                        changeUUID[:],
			ChangingItems:                     nil, // uses pre-stored items from AddPlannedChanges
			BlastRadiusConfigOverride:         analysisConfig.BlastRadiusConfig,
			RoutineChangesConfigOverride:      analysisConfig.RoutineChangesConfig,
			GithubOrganisationProfileOverride: analysisConfig.GithubOrgProfile,
			Knowledge:                         analysisConfig.KnowledgeFiles,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to start change analysis",
		}
	}

	app, _ = strings.CutSuffix(app, "/")
	changeUrl := fmt.Sprintf("%v/changes/%v?utm_source=cli&cli_version=%v", app, changeUUID, tracing.Version())
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("Change analysis started")
	fmt.Println(changeUrl)

	if viper.GetBool("wait") {
		log.WithContext(ctx).WithFields(lf).Info("Waiting for analysis to complete")
		return waitForChangeAnalysis(ctx, client, changeUUID, lf)
	}

	return nil
}

func init() {
	changesCmd.AddCommand(startAnalysisCmd)

	addAPIFlags(startAnalysisCmd)
	addChangeUuidFlags(startAnalysisCmd)
	addAnalysisFlags(startAnalysisCmd)

	startAnalysisCmd.PersistentFlags().Bool("wait", false, "Wait for analysis to complete before returning.")
}
