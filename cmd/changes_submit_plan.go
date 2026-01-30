package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/durationpb"
)

// submitPlanCmd represents the submit-plan command
var submitPlanCmd = &cobra.Command{
	Use:   "submit-plan [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] FILE [FILE ...]",
	Short: "Creates a new Change from a given terraform plan file",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return flagError{fmt.Sprintf("no plan files specified\n\n%v", cmd.UsageString())}
		}
		for _, f := range args {
			_, err := os.Stat(f)
			if err != nil {
				return err
			}
		}
		return nil
	},
	PreRun: PreRunSetup,
	RunE:   SubmitPlan,
}

type TfData struct {
	Address string
	Type    string
	Values  map[string]any
}

func changeTitle(ctx context.Context, arg string) string {
	if arg != "" {
		// easy, return the user's choice
		return arg
	}

	describeBytes, err := exec.CommandContext(ctx, "git", "describe", "--long").Output()
	describe := strings.TrimSpace(string(describeBytes))
	if err != nil {
		log.WithError(err).Trace("failed to run 'git describe' for default title")
		describe, err = os.Getwd()
		if err != nil {
			log.WithError(err).Trace("failed to get current directory for default title")
			describe = "unknown"
		}
	}

	u, err := user.Current()
	var username string
	if err != nil {
		log.WithError(err).Trace("failed to get current user for default title")
		username = "unknown"
	} else {
		username = u.Username
	}

	result := fmt.Sprintf("Deployment from %v by %v", describe, username)
	log.WithField("generated-title", result).Debug("Using default title")
	return result
}

func tryLoadText(ctx context.Context, fileName string) string {
	if fileName == "" {
		return ""
	}

	bytes, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("file", fileName).Warn("Failed to read file")
		return ""
	}

	return strings.TrimSpace(string(bytes))
}

func createBlastRadiusConfig(maxDepth, maxItems int32, maxTime, changeAnalysisTargetDuration time.Duration) (*sdp.BlastRadiusConfig, error) {
	var blastRadiusConfigOverride *sdp.BlastRadiusConfig
	if maxDepth > 0 || maxItems > 0 || maxTime > 0 || changeAnalysisTargetDuration > 0 {
		blastRadiusConfigOverride = &sdp.BlastRadiusConfig{
			MaxItems:  maxItems,
			LinkDepth: maxDepth,
		}
		// this is for backward compatibility, remove in a future release
		if maxTime > 0 {
			// we convert the maxTime to changeAnalysisTargetDuration, this means multiplying the (blast radius calculation timeout) maxTime by 1.5
			// eg 10 minute max (blast radius calculation) -> 15 minute target duration
			blastRadiusConfigOverride.ChangeAnalysisTargetDuration = durationpb.New(time.Duration(float64(maxTime) * 1.5))
		}
		// Add changeAnalysisTargetDuration if specified
		if changeAnalysisTargetDuration > 0 {
			blastRadiusConfigOverride.ChangeAnalysisTargetDuration = durationpb.New(changeAnalysisTargetDuration)
		}
	}

	// validate the ChangeAnalysisTargetDuration
	if blastRadiusConfigOverride != nil && blastRadiusConfigOverride.GetChangeAnalysisTargetDuration() != nil {
		changeAnalysisTargetDuration = blastRadiusConfigOverride.GetChangeAnalysisTargetDuration().AsDuration()
		if changeAnalysisTargetDuration < 1*time.Minute || changeAnalysisTargetDuration > 30*time.Minute {
			return nil, flagError{"--change-analysis-target-duration must be between 1 minute and 30 minutes"}
		}
	}

	return blastRadiusConfigOverride, nil
}

func SubmitPlan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	app := viper.GetString("app")

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write", "sources:read"}, nil)
	if err != nil {
		return err
	}

	lf := log.Fields{}

	// Detect the repository URL if it wasn't provided
	repoUrl := viper.GetString("repo")
	if repoUrl == "" {
		repoUrl, err = DetectRepoURL(AllDetectors)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Debug("Failed to detect repository URL. Use the --repo flag to specify it manually if you require it")
		}
	}
	scope := tfutils.RepoToScope(repoUrl)

	fileWord := "file"
	if len(args) > 1 {
		fileWord = "files"
	}

	log.WithContext(ctx).Infof("Reading %v plan %v", len(args), fileWord)

	plannedChanges := make([]*sdp.MappedItemDiff, 0)

	for _, f := range args {
		lf["file"] = f
		result, err := tfutils.MappedItemDiffsFromPlanFile(ctx, f, scope, lf)
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Error parsing terraform plan",
			}
		}
		plannedChanges = append(plannedChanges, result.GetItemDiffs()...)
	}
	delete(lf, "file")

	client := AuthenticatedChangesClient(ctx, oi)
	changeUUID, err := getChangeUUIDAndCheckStatus(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), false)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed searching for existing changes",
		}
	}

	title := changeTitle(ctx, viper.GetString("title"))
	tfPlanOutput := tryLoadText(ctx, viper.GetString("terraform-plan-output"))
	codeChangesOutput := tryLoadText(ctx, viper.GetString("code-changes-diff"))

	enrichedTags, err := parseTagsArgument()
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to parse tags",
		}
	}

	labels, err := parseLabelsArgument()
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to parse labels",
		}
	}
	properties := &sdp.ChangeProperties{
		Title:        title,
		Description:  viper.GetString("description"),
		TicketLink:   viper.GetString("ticket-link"),
		Owner:        viper.GetString("owner"),
		RawPlan:      tfPlanOutput,
		CodeChanges:  codeChangesOutput,
		Repo:         repoUrl,
		EnrichedTags: enrichedTags,
		Labels:       labels,
	}

	if changeUUID == uuid.Nil {
		log.WithContext(ctx).WithFields(lf).Debug("Creating a new change")

		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: properties,
			},
		})
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to create change",
			}
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to read change id",
			}
		}

		changeUUID = *maybeChangeUuid
		lf["change"] = changeUUID
		log.WithContext(ctx).WithFields(lf).Info("Created a new change")
	} else {
		lf["change"] = changeUUID
		log.WithContext(ctx).WithFields(lf).Debug("Updating an existing change")

		_, err := client.UpdateChange(ctx, &connect.Request[sdp.UpdateChangeRequest]{
			Msg: &sdp.UpdateChangeRequest{
				UUID:       changeUUID[:],
				Properties: properties,
			},
		})
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Failed to update change",
			}
		}

		log.WithContext(ctx).WithFields(lf).Info("Re-using change")
	}

	// Set up the blast radius preset if specified
	maxDepth := viper.GetInt32("blast-radius-link-depth")
	maxItems := viper.GetInt32("blast-radius-max-items")
	maxTime := viper.GetDuration("blast-radius-max-time")
	changeAnalysisTargetDuration := viper.GetDuration("change-analysis-target-duration")

	blastRadiusConfigOverride, err := createBlastRadiusConfig(maxDepth, maxItems, maxTime, changeAnalysisTargetDuration)
	if err != nil {
		return err
	}

	// setup the signal config if specified, or found in the default location
	// order of precedence: flag > default config file
	signalConfigPath := viper.GetString("signal-config")
	signalConfigOverride, err := checkForAndLoadSignalConfigFile(ctx, lf, signalConfigPath)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to load signal config",
		}
	}

	var githubOrganisationProfileOverride *sdp.GithubOrganisationProfile
	var routineChangesConfigOverride *sdp.RoutineChangesConfig
	if signalConfigOverride != nil {
		githubOrganisationProfileOverride = signalConfigOverride.GithubOrganisationProfile
		routineChangesConfigOverride = signalConfigOverride.RoutineChangesConfig
	}

	_, err = client.StartChangeAnalysis(ctx, &connect.Request[sdp.StartChangeAnalysisRequest]{
		Msg: &sdp.StartChangeAnalysisRequest{
			ChangeUUID:                        changeUUID[:],
			ChangingItems:                     plannedChanges,
			BlastRadiusConfigOverride:         blastRadiusConfigOverride,
			RoutineChangesConfigOverride:      routineChangesConfigOverride,
			GithubOrganisationProfileOverride: githubOrganisationProfileOverride,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to start change analysis",
		}
	}

	app, _ = strings.CutSuffix(app, "/")
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", app, changeUUID)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("Change ready")
	fmt.Println(changeUrl)

	return nil
}

func loadSignalConfigFile(signalConfigPath string) (*sdp.SignalConfigFile, error) {
	// check if the file exists
	_, err := os.Stat(signalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("signal config file %q does not exist: %w", signalConfigPath, err)
	}

	// read the file
	signalConfig, err := os.ReadFile(signalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read signal config file %q: %w", signalConfigPath, err)
	}

	signalConfigOverride, err := sdp.YamlStringToSignalConfig(string(signalConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse signal config file %q: %w", signalConfigPath, err)
	}

	return signalConfigOverride, nil
}

// order of precedence: flag > default config file
func checkForAndLoadSignalConfigFile(ctx context.Context, lf log.Fields, manualPath string) (*sdp.SignalConfigFile, error) {
	foundPath := ""
	if manualPath != "" {
		_, err := os.Stat(manualPath)
		if err == nil {
			// we found the file
			foundPath = manualPath
		} else {
			// the specified file does not exist
			// hard fail
			lf["signalConfig"] = manualPath
			err = fmt.Errorf("signal config file does not exist: %w", err)
			return nil, err
		}
	}
	// let's look for the default files
	// yaml
	if foundPath == "" {
		_, err := os.Stat(".overmind/signal-config.yaml")
		if err == nil {
			// we found the file
			foundPath = ".overmind/signal-config.yaml"
		}
	}
	// yml
	if foundPath == "" {
		_, err := os.Stat(".overmind/signal-config.yml")
		if err == nil {
			// we found the file
			foundPath = ".overmind/signal-config.yml"
		}
	}

	if foundPath != "" {
		// we found a file, load it
		lf["signalConfig"] = foundPath
		log.WithContext(ctx).WithFields(lf).Info("Loading signal config")
		signalConfigOverride, err := loadSignalConfigFile(foundPath)
		if err != nil {
			return nil, err
		}
		return signalConfigOverride, nil
	}
	// we didn't find any files, thats ok
	return nil, nil
}

func init() {
	changesCmd.AddCommand(submitPlanCmd)

	addAPIFlags(submitPlanCmd)
	addChangeCreationFlags(submitPlanCmd)

	submitPlanCmd.PersistentFlags().String("frontend", "", "The frontend base URL")
	_ = submitPlanCmd.PersistentFlags().MarkDeprecated("frontend", "This flag is no longer used and will be removed in a future release. Use the '--app' flag instead.") // MarkDeprecated only errors if the flag doesn't exist, we fall back to using app

	submitPlanCmd.PersistentFlags().Int32("blast-radius-link-depth", 0, "Used in combination with '--blast-radius-max-items' to customise how many levels are traversed when calculating the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")
	submitPlanCmd.PersistentFlags().Int32("blast-radius-max-items", 0, "Used in combination with '--blast-radius-link-depth' to customise how many items are included in the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")

	submitPlanCmd.PersistentFlags().Duration("blast-radius-max-time", 0, "Maximum time duration for blast radius calculation (e.g., '5m', '15m', '30m'). When the time limit is reached, the analysis continues with risks identified up to that point. Defaults to the account level settings (QUICK: 10m, DETAILED: 15m, FULL: 30m). Valid range: 1m to 30m.")
	_ = submitPlanCmd.PersistentFlags().MarkDeprecated("blast-radius-max-time", "This flag is no longer used and will be removed in a future release. Use the '--change-analysis-target-duration' flag instead.")
	submitPlanCmd.PersistentFlags().Duration("change-analysis-target-duration", 0, "Target duration for change analysis planning (e.g., '5m', '15m', '30m'). This is NOT a hard deadline - the blast radius phase uses 67% of this target to stop gracefully. The job can run slightly past this target and is only hard-stopped at 30 minutes. Defaults to the account level settings (QUICK: 10m, DETAILED: 15m, FULL: 30m). Valid range: 1m to 30m.")
	submitPlanCmd.PersistentFlags().String("auto-tag-rules", "", "The path to the auto-tag rules file. If not provided, it will check the default location which is '.overmind/auto-tag-rules.yaml'. If no rules are found locally, the rules configured through the UI are used.")
	submitPlanCmd.PersistentFlags().String("signal-config", "", "The path to the signal config file. If not provided, it will check the default location which is '.overmind/signal-config.yaml'. If no config is found locally, the config configured through the UI is used.")
}
