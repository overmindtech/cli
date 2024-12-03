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
	"github.com/overmindtech/sdp-go"
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
	}
	username = u.Username

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

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"}, nil)
	if err != nil {
		return err
	}

	fileWord := "file"
	if len(args) > 1 {
		fileWord = "files"
	}

	log.WithContext(ctx).Infof("Reading %v plan %v", len(args), fileWord)

	plannedChanges := make([]*sdp.MappedItemDiff, 0)

	lf := log.Fields{}
	for _, f := range args {
		lf["file"] = f
		result, err := tfutils.MappedItemDiffsFromPlanFile(ctx, f, lf)
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
	// Detect the repository URL if it wasn't provided
	repoUrl := viper.GetString("repo")
	if repoUrl == "" {
		repoUrl, err = DetectRepoURL(AllDetectors)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Debug("Failed to detect repository URL. Use the --repo flag to specify it manually if you require it")
		}
	}
	tags, err := parseTagsArgument()
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to parse tags",
		}
	}
	properties := &sdp.ChangeProperties{
		Title:       title,
		Description: viper.GetString("description"),
		TicketLink:  viper.GetString("ticket-link"),
		Owner:       viper.GetString("owner"),
		RawPlan:     tfPlanOutput,
		CodeChanges: codeChangesOutput,
		Repo:        repoUrl,
		Tags:        tags,
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

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:                changeUuid[:],
			ChangingItems:             plannedChanges,
			BlastRadiusConfigOverride: blastRadiusConfigOverride,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to update planned changes",
		}
	}

	last_log := time.Now()
	first_log := true
	for resultStream.Receive() {
		msg := resultStream.Msg()

		// log the first message and at most every 250ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.GetState() != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(lf).WithField("msg", msg).Info("Status update")
			last_log = time.Now()
			first_log = false
		}
	}
	if resultStream.Err() != nil {
		return loggedError{
			err:     resultStream.Err(),
			fields:  lf,
			message: "Error streaming results",
		}
	}

	app, _ = strings.CutSuffix(app, "/")
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", app, changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("Change ready")
	fmt.Println(changeUrl)

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("")
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to get updated change",
		}
	}

	for _, a := range fetchResponse.Msg.GetChange().GetProperties().GetAffectedAppsUUID() {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).WithField("app", a).Error("Received invalid app uuid")
			continue
		}
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"change-url": changeUrl,
			"app":        appUuid,
			"app-url":    fmt.Sprintf("%v/apps/%v", app, appUuid),
		}).Info("Affected app")
	}

	return nil
}

func init() {
	changesCmd.AddCommand(submitPlanCmd)

	addAPIFlags(submitPlanCmd)
	addChangeCreationFlags(submitPlanCmd)

	submitPlanCmd.PersistentFlags().String("frontend", "", "The frontend base URL")
	_ = submitPlanCmd.PersistentFlags().MarkDeprecated("frontend", "This flag is no longer used and will be removed in a future release. Use the '--app' flag instead.") // MarkDeprecated only errors if the flag doesn't exist, we fall back to using app

	submitPlanCmd.PersistentFlags().Int32("blast-radius-link-depth", 0, "Used in combination with '--blast-radius-max-items' to customise how many levels are traversed when calculating the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")
	submitPlanCmd.PersistentFlags().Int32("blast-radius-max-items", 0, "Used in combination with '--blast-radius-link-depth' to customise how many items are included in the blast radius. Larger numbers will result in a more comprehensive blast radius, but may take longer to calculate. Defaults to the account level settings.")
}
