package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tfutils"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func changeTitle(arg string) string {
	if arg != "" {
		// easy, return the user's choice
		return arg
	}

	describeBytes, err := exec.Command("git", "describe", "--long").Output()
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
	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), false)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed searching for existing changes",
		}
	}

	title := changeTitle(viper.GetString("title"))
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

	properties := &sdp.ChangeProperties{
		Title:        title,
		Description:  viper.GetString("description"),
		TicketLink:   viper.GetString("ticket-link"),
		Owner:        viper.GetString("owner"),
		RawPlan:      tfPlanOutput,
		CodeChanges:  codeChangesOutput,
		Repo:         repoUrl,
		EnrichedTags: enrichedTags,
	}

	if changeUuid == uuid.Nil {
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

		changeUuid = *maybeChangeUuid
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("Created a new change")
	} else {
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Debug("Updating an existing change")

		_, err := client.UpdateChange(ctx, &connect.Request[sdp.UpdateChangeRequest]{
			Msg: &sdp.UpdateChangeRequest{
				UUID:       changeUuid[:],
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
	var blastRadiusConfigOverride *sdp.BlastRadiusConfig
	if maxDepth > 0 || maxItems > 0 {
		blastRadiusConfigOverride = &sdp.BlastRadiusConfig{
			MaxItems:  maxItems,
			LinkDepth: maxDepth,
		}
	}

	// Set up the local auto-tag rules if specified, or found in the default location
	// order of precedence: flag > default config file
	autoTagRulesPath := viper.GetString("auto-tag-rules")
	autoTaggingRulesOverride, err := checkForAndLoadAutoTagRulesFile(ctx, lf, autoTagRulesPath)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to load auto-tag rules",
		}
	}

	_, err = client.StartChangeAnalysis(ctx, &connect.Request[sdp.StartChangeAnalysisRequest]{
		Msg: &sdp.StartChangeAnalysisRequest{
			ChangeUUID:                changeUuid[:],
			ChangingItems:             plannedChanges,
			BlastRadiusConfigOverride: blastRadiusConfigOverride,
			AutoTaggingRulesOverride:  autoTaggingRulesOverride,
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
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", app, changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("Change ready")
	fmt.Println(changeUrl)

	return nil
}

func loadAutoTagRulesFile(autoTagRulesPath string) ([]*sdp.RuleProperties, error) {
	// check if the file exists
	_, err := os.Stat(autoTagRulesPath)
	if err != nil {
		return nil, fmt.Errorf("Auto-tag rules file %q does not exist: %w", autoTagRulesPath, err)
	}
	// read the file
	autoTagRules, err := os.ReadFile(autoTagRulesPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read auto-tag rules file %q: %w", autoTagRulesPath, err)
	}
	autoTaggingRulesOverride, err := sdp.YamlStringToRuleProperties(string(autoTagRules))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse auto-tag rules file %q: %w", autoTagRulesPath, err)
	}
	if len(autoTaggingRulesOverride) > 10 {
		return nil, errors.New("Auto-tag rules file contains more than 10 rules")
	}
	return autoTaggingRulesOverride, nil
}

// order of precedence: flag > default config file
func checkForAndLoadAutoTagRulesFile(ctx context.Context, lf log.Fields, manualPath string) ([]*sdp.RuleProperties, error) {
	foundPath := ""
	if manualPath != "" {
		_, err := os.Stat(manualPath)
		if err == nil {
			// we found the file
			foundPath = manualPath
		} else {
			// the specified file does not exist
			// hard fail
			lf["autoTagRules"] = manualPath
			err = fmt.Errorf("Auto-tag rules file does not exist: %w", err)
			return nil, err
		}
	}
	// lets look for the default files
	if foundPath == "" {
		_, err := os.Stat(".overmind/auto-tag-rules.yaml")
		if err == nil {
			// we found the file
			foundPath = ".overmind/auto-tag-rules.yaml"
		}
	}
	if foundPath == "" {
		_, err := os.Stat(".overmind/auto-tag-rules.yml")
		if err == nil {
			// we found the file
			foundPath = ".overmind/auto-tag-rules.yml"
		}
	}

	if foundPath != "" {
		// we found a file, load it
		lf["autoTagRules"] = foundPath
		log.WithContext(ctx).WithFields(lf).Info("Loading auto-tag rules")
		autoTaggingRulesOverride, err := loadAutoTagRulesFile(foundPath)
		if err != nil {
			return nil, err
		}
		return autoTaggingRulesOverride, nil
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
	submitPlanCmd.PersistentFlags().String("auto-tag-rules", "", "The path to the auto-tag rules file. If not provided, it will check the default location which is '.overmind/auto-tag-rules.yaml'. If no rules are found locally, the rules configured through the UI are used.")
}
