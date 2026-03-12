package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/knowledge"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// This file contains re-usable sets of flags that should be used when creating
// commands

// Adds flags for selecting a change by UUID, frontend URL or ticket link
func addChangeUuidFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("change", "", "The frontend URL of the change to get")
	cmd.PersistentFlags().String("ticket-link", "", "Link to the ticket for this change.")
	cmd.PersistentFlags().String("uuid", "", "The UUID of the change that should be displayed.")
	cmd.MarkFlagsMutuallyExclusive("change", "ticket-link", "uuid")
}

// Adds flags that should be present when creating a change
func addChangeCreationFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("title", "", "Short title for this change. If this is not specified, overmind will try to come up with one for you.")
	cmd.PersistentFlags().String("description", "", "Quick description of the change.")
	cmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change. Usually this would be the link to something like the pull request, since the CLI uses this as a unique identifier for the change, meaning that multiple runs with the same ticket link will update the same change.")
	cmd.PersistentFlags().String("owner", "", "The owner of this change.")
	cmd.PersistentFlags().String("repo", "", "The repository URL that this change should be linked to. This will be automatically detected is possible from the Git config or CI environment.")
	cmd.PersistentFlags().String("terraform-plan-output", "", "Filename of cached terraform plan output for this change.")
	cmd.PersistentFlags().String("code-changes-diff", "", "Filename of the code diff of this change.")
	cmd.PersistentFlags().StringSlice("tags", []string{}, "Tags to apply to this change, these should be specified in key=value format. Multiple tags can be specified by repeating the flag or using a comma separated list.")
	// ENG-1985, disabled until we decide how manual labels and manual tags should be handled.
	// cmd.PersistentFlags().StringSlice("labels", []string{}, "Labels to apply to this change, these should be specified in name=color format where color is a hex code (e.g., FF0000 or #FF0000). Multiple labels can be specified by repeating the flag or using a comma separated list.")
}

func parseTagsArgument() (*sdp.EnrichedTags, error) {
	tags := map[string]string{}
	// get into key pair
	for _, tag := range viper.GetStringSlice("tags") {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format: %s", tag)
		}
		tags[parts[0]] = parts[1]
	}
	// put into enriched tags
	enrichedTags := &sdp.EnrichedTags{
		TagValue: make(map[string]*sdp.TagValue),
	}
	for key, value := range tags {
		enrichedTags.TagValue[key] = &sdp.TagValue{
			Value: &sdp.TagValue_UserTagValue{
				UserTagValue: &sdp.UserTagValue{
					Value: value,
				},
			},
		}
	}
	return enrichedTags, nil
}

func parseLabelsArgument() ([]*sdp.Label, error) {
	labels := make([]*sdp.Label, 0)
	for _, label := range viper.GetStringSlice("labels") {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %s (expected name=color)", label)
		}
		if parts[0] == "" {
			return nil, fmt.Errorf("invalid label format: %s (label name cannot be empty)", label)
		}

		// Normalise colour: strip leading # if present, validate, then add # back
		colour := strings.TrimPrefix(parts[1], "#")
		if colour == "" {
			return nil, fmt.Errorf("invalid colour format: %s (colour cannot be empty)", parts[1])
		}

		// Validate it's exactly 6 hex digits
		if len(colour) != 6 {
			return nil, fmt.Errorf("invalid colour format: %s (must be 6 hex digits, got %d)", parts[1], len(colour))
		}

		// Validate all characters are valid hex digits
		if _, err := strconv.ParseUint(colour, 16, 64); err != nil {
			return nil, fmt.Errorf("invalid colour format: %s (must be valid hex digits)", parts[1])
		}

		// Normalise to canonical form: always #rrggbb
		normalisedColour := "#" + strings.ToUpper(colour)

		labels = append(labels, &sdp.Label{
			Name:   parts[0],
			Colour: normalisedColour,
			Type:   sdp.LabelType_LABEL_TYPE_USER,
		})
	}
	return labels, nil
}

// Adds common flags to API commands e.g. timeout
func addAPIFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("timeout", "31m", "How long to wait for responses")
	cmd.PersistentFlags().String("app", "https://app.overmind.tech", "The overmind instance to connect to.")
}

// Adds terraform-related flags to a command
func addTerraformBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("reset-stored-config", false, "[deprecated: this is now autoconfigured from local terraform files] Set this to reset the sources config stored in Overmind and input fresh values.")
	cmd.PersistentFlags().String("aws-config", "", "[deprecated: this is now autoconfigured from local terraform files] The chosen AWS config method, best set through the initial wizard when running the CLI. Options: 'profile_input', 'aws_profile', 'defaults', 'managed'.")
	cmd.PersistentFlags().String("aws-profile", "", "[deprecated: this is now autoconfigured from local terraform files] Set this to the name of the AWS profile to use.")
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("reset-stored-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-profile"))
	cmd.PersistentFlags().Bool("only-use-managed-sources", false, "Set this to skip local autoconfiguration and only use the managed sources as configured in Overmind.")
}

// Adds analysis-related flags (blast radius config, signal config) to a command.
// These flags are shared between submit-plan and start-analysis commands.
func addAnalysisFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Int32("blast-radius-link-depth", 0, "Used in combination with '--blast-radius-max-items' to customise how many levels are traversed when calculating the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")
	cmd.PersistentFlags().Int32("blast-radius-max-items", 0, "Used in combination with '--blast-radius-link-depth' to customise how many items are included in the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")
	cmd.PersistentFlags().Duration("blast-radius-max-time", 0, "Maximum time duration for blast radius calculation (e.g., '5m', '15m', '30m'). When the time limit is reached, the analysis continues with risks identified up to that point. Defaults to the account level settings (QUICK: 10m, DETAILED: 15m, FULL: 30m). Valid range: 1m to 30m.")
	cobra.CheckErr(cmd.PersistentFlags().MarkDeprecated("blast-radius-max-time", "This flag is no longer used and will be removed in a future release. Use the '--change-analysis-target-duration' flag instead."))
	cmd.PersistentFlags().Duration("change-analysis-target-duration", 0, "Target duration for change analysis planning (e.g., '5m', '15m', '30m'). This is NOT a hard deadline - the blast radius phase uses 67% of this target to stop gracefully. The job can run slightly past this target and is only hard-stopped at 30 minutes. Defaults to the account level settings (QUICK: 10m, DETAILED: 15m, FULL: 30m). Valid range: 1m to 30m.")
	cmd.PersistentFlags().String("signal-config", "", "The path to the signal config file. If not provided, it will check the default location which is '.overmind/signal-config.yaml'. If no config is found locally, the config configured through the UI is used.")
	cmd.PersistentFlags().Bool("comment", false, "Request the GitHub App to post analysis results as a PR comment. Requires the account to have the Overmind GitHub App installed with pull_requests:write.")
}

// AnalysisConfig holds all the configuration needed to start change analysis.
type AnalysisConfig struct {
	BlastRadiusConfig *sdp.BlastRadiusConfig
	RoutineChangesConfig *sdp.RoutineChangesConfig
	GithubOrgProfile *sdp.GithubOrganisationProfile
	KnowledgeFiles []*sdp.Knowledge
}

// buildAnalysisConfig reads viper flags and builds the analysis configuration
// used by StartChangeAnalysis. This includes blast radius config, routine changes
// config, github org profile, and knowledge files.
func buildAnalysisConfig(ctx context.Context, lf log.Fields) (*AnalysisConfig, error) {
	maxDepth := viper.GetInt32("blast-radius-link-depth")
	maxItems := viper.GetInt32("blast-radius-max-items")
	maxTime := viper.GetDuration("blast-radius-max-time")
	changeAnalysisTargetDuration := viper.GetDuration("change-analysis-target-duration")

	blastRadiusConfig, err := createBlastRadiusConfig(maxDepth, maxItems, maxTime, changeAnalysisTargetDuration)
	if err != nil {
		return nil, err
	}

	signalConfigPath := viper.GetString("signal-config")
	signalConfigOverride, err := checkForAndLoadSignalConfigFile(ctx, lf, signalConfigPath)
	if err != nil {
		return nil, loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to load signal config",
		}
	}

	var githubOrgProfile *sdp.GithubOrganisationProfile
	var routineChangesConfig *sdp.RoutineChangesConfig
	if signalConfigOverride != nil {
		githubOrgProfile = signalConfigOverride.GithubOrganisationProfile
		routineChangesConfig = signalConfigOverride.RoutineChangesConfig
	}

	knowledgeDir := knowledge.FindKnowledgeDir(".")
	knowledgeFiles := knowledge.DiscoverAndConvert(ctx, knowledgeDir)

	return &AnalysisConfig{
		BlastRadiusConfig:    blastRadiusConfig,
		RoutineChangesConfig: routineChangesConfig,
		GithubOrgProfile:     githubOrgProfile,
		KnowledgeFiles:       knowledgeFiles,
	}, nil
}

// waitForChangeAnalysis polls the change until analysis reaches a terminal status
// (STATUS_DONE, STATUS_SKIPPED, or STATUS_ERROR). It returns nil on successful
// completion, or an error if analysis failed or was cancelled.
func waitForChangeAnalysis(ctx context.Context, client sdpconnect.ChangesServiceClient, changeUUID uuid.UUID, lf log.Fields) error {
	for {
		changeRes, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
			Msg: &sdp.GetChangeRequest{
				UUID: changeUUID[:],
			},
		})
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "failed to get change",
			}
		}
		if changeRes.Msg == nil || changeRes.Msg.GetChange() == nil {
			return loggedError{
				err:     fmt.Errorf("unexpected nil response from GetChange"),
				fields:  lf,
				message: "failed to get change",
			}
		}

		ch := changeRes.Msg.GetChange()
		md := ch.GetMetadata()
		if md == nil || md.GetChangeAnalysisStatus() == nil {
			return loggedError{
				err:     fmt.Errorf("change metadata or change analysis status is nil"),
				fields:  lf,
				message: "failed to get change analysis status",
			}
		}

		status := md.GetChangeAnalysisStatus().GetStatus()
		switch status {
		case sdp.ChangeAnalysisStatus_STATUS_DONE, sdp.ChangeAnalysisStatus_STATUS_SKIPPED:
			log.WithContext(ctx).WithFields(lf).WithField("status", status.String()).Info("Change analysis complete")
			return nil
		case sdp.ChangeAnalysisStatus_STATUS_ERROR:
			return loggedError{
				err:     fmt.Errorf("change analysis completed with error status"),
				fields:  lf,
				message: "change analysis failed",
			}
		case sdp.ChangeAnalysisStatus_STATUS_UNSPECIFIED, sdp.ChangeAnalysisStatus_STATUS_INPROGRESS:
			log.WithContext(ctx).WithFields(lf).WithField("status", status.String()).Info("Waiting for change analysis to complete")
		}

		time.Sleep(3 * time.Second)
		if ctx.Err() != nil {
			return loggedError{
				err:     ctx.Err(),
				fields:  lf,
				message: "context cancelled",
			}
		}
	}
}
