package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
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

// maskAllData masks every entry in attributes as redacted
func maskAllData(attributes map[string]any) map[string]any {
	for k, v := range attributes {
		if mv, ok := v.(map[string]any); ok {
			attributes[k] = maskAllData(mv)
		} else {
			attributes[k] = "REDACTED"
		}
	}
	return attributes
}

// maskSensitiveData masks every entry in attributes that is set to true in sensitive. returns the redacted attributes
func maskSensitiveData(attributes, sensitive any) any {
	if sensitive == true {
		return "REDACTED"
	} else if sensitiveMap, ok := sensitive.(map[string]any); ok {
		if attributesMap, ok := attributes.(map[string]any); ok {
			result := map[string]any{}
			for k, v := range attributesMap {
				result[k] = maskSensitiveData(v, sensitiveMap[k])
			}
			return result
		} else {
			return "REDACTED (type mismatch)"
		}
	} else if sensitiveArr, ok := sensitive.([]any); ok {
		if attributesArr, ok := attributes.([]any); ok {
			if len(sensitiveArr) != len(attributesArr) {
				return "REDACTED (len mismatch)"
			}
			result := make([]any, len(attributesArr))
			for i, v := range attributesArr {
				result[i] = maskSensitiveData(v, sensitiveArr[i])
			}
			return result
		} else {
			return "REDACTED (type mismatch)"
		}
	}
	return attributes
}

func itemAttributesFromResourceChangeData(attributesMsg, sensitiveMsg json.RawMessage) (*sdp.ItemAttributes, error) {
	var attributes map[string]any
	err := json.Unmarshal(attributesMsg, &attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse attributes: %w", err)
	}

	// sensitiveMsg can be a bool or a map[string]any
	var isSensitive bool
	err = json.Unmarshal(sensitiveMsg, &isSensitive)
	if err == nil && isSensitive {
		attributes = maskAllData(attributes)
	} else if err != nil {
		// only try parsing as map if parsing as bool failed
		var sensitive map[string]any
		err = json.Unmarshal(sensitiveMsg, &sensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sensitive: %w", err)
		}
		attributes = maskSensitiveData(attributes, sensitive).(map[string]any)
	}

	return sdp.ToAttributesSorted(attributes)
}

// Converts a ResourceChange form a terraform plan to an ItemDiff in SDP format.
// These items will use the scope `terraform_plan` since we haven't mapped them
// to an actual item in the infrastructure yet
func itemDiffFromResourceChange(resourceChange ResourceChange) (*sdp.ItemDiff, error) {
	status := sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED

	if slices.Equal(resourceChange.Change.Actions, []string{"no-op"}) || slices.Equal(resourceChange.Change.Actions, []string{"read"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"create"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"update"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"delete", "create"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"create", "delete"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"delete"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED
	}

	beforeAttributes, err := itemAttributesFromResourceChangeData(resourceChange.Change.Before, resourceChange.Change.BeforeSensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse before attributes: %w", err)
	}
	afterAttributes, err := itemAttributesFromResourceChangeData(resourceChange.Change.After, resourceChange.Change.AfterSensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse after attributes: %w", err)
	}

	result := &sdp.ItemDiff{
		// Item: filled in by item mapping in UpdatePlannedChanges
		Status: status,
	}

	// shorten the address by removing the type prefix if and only if it is the
	// first part. Longer terraform addresses created in modules will not be
	// shortened to avoid confusion.
	trimmedAddress, _ := strings.CutPrefix(resourceChange.Address, fmt.Sprintf("%v.", resourceChange.Type))

	if beforeAttributes != nil {
		result.Before = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_name",
			Attributes:      beforeAttributes,
			Scope:           "terraform_plan",
		}

		err = result.GetBefore().GetAttributes().Set("terraform_name", trimmedAddress)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_name '%v' on before attributes: %w", trimmedAddress, err))
		}

		err = result.GetBefore().GetAttributes().Set("terraform_address", resourceChange.Address)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_address of type %T (%v) on before attributes: %w", resourceChange.Address, resourceChange.Address, err))
		}
	}

	if afterAttributes != nil {
		result.After = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_name",
			Attributes:      afterAttributes,
			Scope:           "terraform_plan",
		}

		err = result.GetAfter().GetAttributes().Set("terraform_name", trimmedAddress)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_name '%v' on after attributes: %w", trimmedAddress, err))
		}

		err = result.GetAfter().GetAttributes().Set("terraform_address", resourceChange.Address)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_address of type %T (%v) on after attributes: %w", resourceChange.Address, resourceChange.Address, err))
		}
	}

	return result, nil
}

type plannedChangeGroups struct {
	supported   map[string][]*sdp.MappedItemDiff
	unsupported map[string][]*sdp.MappedItemDiff
}

func (g *plannedChangeGroups) NumUnsupportedChanges() int {
	num := 0

	for _, v := range g.unsupported {
		num += len(v)
	}

	return num
}

func (g *plannedChangeGroups) NumSupportedChanges() int {
	num := 0

	for _, v := range g.supported {
		num += len(v)
	}

	return num
}

func (g *plannedChangeGroups) MappedItemDiffs() []*sdp.MappedItemDiff {
	mappedItemDiffs := make([]*sdp.MappedItemDiff, 0)

	for _, v := range g.supported {
		mappedItemDiffs = append(mappedItemDiffs, v...)
	}

	for _, v := range g.unsupported {
		mappedItemDiffs = append(mappedItemDiffs, v...)
	}

	return mappedItemDiffs
}

// Add the specified item to the appropriate type group in the supported or unsupported section, based of whether it has a mapping query
func (g *plannedChangeGroups) Add(typ string, item *sdp.MappedItemDiff) {
	groups := g.supported
	if item.GetMappingQuery() == nil {
		groups = g.unsupported
	}
	list, ok := groups[typ]
	if !ok {
		list = make([]*sdp.MappedItemDiff, 0)
	}
	groups[typ] = append(list, item)
}

// Checks if the supplied JSON bytes are a state file. It's a common  mistake to
// pass a state file to Overmind rather than a plan file since the commands to
// create them are similar
func isStateFile(bytes []byte) bool {
	fields := make(map[string]interface{})

	err := json.Unmarshal(bytes, &fields)

	if err != nil {
		return false
	}

	if _, exists := fields["values"]; exists {
		return true
	}

	return false
}

// Returns the name of the provider from the config key. If the resource isn't
// in a module, the ProviderConfigKey will be something like "kubernetes",
// however if it's in a module it's be something like
// "module.something:kubernetes". In both scenarios we want to return
// "kubernetes"
func extractProviderNameFromConfigKey(providerConfigKey string) string {
	sections := strings.Split(providerConfigKey, ":")
	return sections[len(sections)-1]
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

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"})
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
		_, mappedItemDiffs, _, err := mappedItemDiffsFromPlanFile(ctx, f, lf)
		if err != nil {
			return loggedError{
				err:     err,
				fields:  lf,
				message: "Error parsing terraform plan",
			}
		}
		plannedChanges = append(plannedChanges, mappedItemDiffs...)
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

	if changeUuid == uuid.Nil {
		log.WithContext(ctx).WithFields(lf).Debug("Creating a new change")
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  viper.GetString("ticket-link"),
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
					RawPlan:     tfPlanOutput,
					CodeChanges: codeChangesOutput,
				},
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
				UUID: changeUuid[:],
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  viper.GetString("ticket-link"),
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
					RawPlan:     tfPlanOutput,
					CodeChanges: codeChangesOutput,
				},
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

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: plannedChanges,
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

	frontend, _ := strings.CutSuffix(viper.GetString("frontend"), "/")
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", frontend, changeUuid)
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
			"app-url":    fmt.Sprintf("%v/apps/%v", frontend, appUuid),
		}).Info("Affected app")
	}

	return nil
}

func init() {
	changesCmd.AddCommand(submitPlanCmd)

	submitPlanCmd.PersistentFlags().String("frontend", "https://app.overmind.tech", "The frontend base URL")

	submitPlanCmd.PersistentFlags().String("title", "", "Short title for this change. If this is not specified, overmind will try to come up with one for you.")
	submitPlanCmd.PersistentFlags().String("description", "", "Quick description of the change.")
	submitPlanCmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change.")
	submitPlanCmd.PersistentFlags().String("owner", "", "The owner of this change.")
	// submitPlanCmd.PersistentFlags().String("cc-emails", "", "A comma-separated list of emails to keep updated with the status of this change.")

	submitPlanCmd.PersistentFlags().String("terraform-plan-output", "", "Filename of cached terraform plan output for this change.")
	submitPlanCmd.PersistentFlags().String("code-changes-diff", "", "Fileame of the code diff of this change.")
}
