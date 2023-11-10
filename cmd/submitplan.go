package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"slices"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/cmd/datamaps"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// submitPlanCmd represents the submit-plan command
var submitPlanCmd = &cobra.Command{
	Use:   "submit-plan [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] FILE [FILE ...]",
	Short: "Creates a new Change from a given terraform plan file",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no plan files specified")
		}
		for _, f := range args {
			_, err := os.Stat(f)
			if err != nil {
				return err
			}
		}
		return nil
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `submit-plan` flags")
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

		exitcode := SubmitPlan(ctx, args, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
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
func maskSensitiveData(attributes, sensitive map[string]any) map[string]any {
	for k, s := range sensitive {
		log.Debugf("checking %v: %v", k, s)
		if mv, ok := s.(map[string]any); ok {
			if sub, ok := attributes[k].(map[string]any); ok {
				attributes[k] = maskSensitiveData(sub, mv)
			}
		} else if arr, ok := s.([]any); ok {
			if sub, ok := attributes[k].([]any); ok {
				if len(arr) != len(sub) {
					attributes[k] = "REDACTED (len mismatch)"
					continue
				}
				for i, v := range arr {
					if v == true {
						sub[i] = "REDACTED"
					}
				}
				attributes[k] = sub
			}
		} else {
			if _, ok := attributes[k]; ok {
				attributes[k] = "REDACTED"
			}
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
		attributes = maskSensitiveData(attributes, sensitive)
	}

	return sdp.ToAttributesSorted(attributes)
}

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

	if beforeAttributes != nil {
		result.Before = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_address",
			Attributes:      beforeAttributes,
			Scope:           "terraform_plan",
		}

		err = result.Before.Attributes.Set("terraform_address", resourceChange.Address)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_address of type %T (%v) on before attributes: %w", resourceChange.Address, resourceChange.Address, err))
		}
	}

	if afterAttributes != nil {
		result.After = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_address",
			Attributes:      afterAttributes,
			Scope:           "terraform_plan",
		}

		err = result.After.Attributes.Set("terraform_address", resourceChange.Address)
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

// Add the specified item to the approapriate type group in the supported or unsupported section, based of whether it has a mapping query
func (g *plannedChangeGroups) Add(typ string, item *sdp.MappedItemDiff) {
	groups := g.supported
	if item.MappingQuery == nil {
		groups = g.unsupported
	}
	list, ok := groups[typ]
	if !ok {
		list = make([]*sdp.MappedItemDiff, 0)
	}
	groups[typ] = append(list, item)
}

func mappedItemDiffsFromPlan(ctx context.Context, fileName string, lf log.Fields) ([]*sdp.MappedItemDiff, error) {
	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return nil, err
	}

	var plan Plan
	err = json.Unmarshal(planJSON, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse '%v': %w", fileName, err)
	}

	plannedChangeGroups := plannedChangeGroups{
		supported:   map[string][]*sdp.MappedItemDiff{},
		unsupported: map[string][]*sdp.MappedItemDiff{},
	}

	// for all managed resources:
	for _, resourceChange := range plan.ResourceChanges {
		if len(resourceChange.Change.Actions) == 0 || resourceChange.Change.Actions[0] == "no-op" || resourceChange.Mode == "data" {
			// skip resources with no changes and data updates
			continue
		}

		itemDiff, err := itemDiffFromResourceChange(resourceChange)
		if err != nil {
			return nil, fmt.Errorf("failed to create item diff for resource change: %w", err)
		}

		awsMappings := datamaps.AwssourceData[resourceChange.Type]
		k8sMappings := datamaps.K8ssourceData[resourceChange.Type]

		mappings := append(awsMappings, k8sMappings...)

		if len(mappings) == 0 {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
			plannedChangeGroups.Add(resourceChange.Type, &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: nil, // unmapped item has no mapping query
			})
			continue
		}

		for _, mapData := range mappings {
			var currentResource *Resource

			// Look for the resource in the prior values first, since this is
			// the *previous* state we're like to be able to find it in the
			// actual infra
			if plan.PriorState.Values != nil {
				currentResource = plan.PriorState.Values.RootModule.DigResource(resourceChange.Address)
			}

			// If we didn't find it, look in the planned values
			if currentResource == nil {
				currentResource = plan.PlannedValues.RootModule.DigResource(resourceChange.Address)
			}

			if currentResource == nil {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", resourceChange.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("Skipping resource without values")
				continue
			}

			query, ok := currentResource.AttributeValues.Dig(mapData.QueryField)
			if !ok {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", resourceChange.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("Skipping resource without query field")
				continue
			}

			// Create the map that variables will pull data from
			dataMap := make(map[string]any)

			// Populate resource values
			dataMap["values"] = currentResource.AttributeValues

			if overmindMappingsOutput, ok := plan.PlannedValues.Outputs["overmind_mappings"]; ok {
				configResource := plan.Config.RootModule.DigResource(resourceChange.Address)

				if configResource == nil {
					log.WithContext(ctx).
						WithFields(lf).
						WithField("terraform-address", resourceChange.Address).
						Debug("Skipping provider mapping for resource without config")
				} else {
					// Look up the provider config key in the mappings
					mappings := make(map[string]map[string]string)

					err = json.Unmarshal(overmindMappingsOutput.Value, &mappings)

					if err != nil {
						log.WithContext(ctx).
							WithFields(lf).
							WithField("terraform-address", resourceChange.Address).
							WithError(err).
							Error("Failed to parse overmind_mappings output")
					} else {
						currentProviderMappings, ok := mappings[configResource.ProviderConfigKey]

						if ok {
							log.WithContext(ctx).
								WithFields(lf).
								WithField("terraform-address", resourceChange.Address).
								WithField("provider-config-key", configResource.ProviderConfigKey).
								Debug("Found provider mappings")

							// We have mappings for this provider, so set them
							// in the `provider_mapping` value
							dataMap["provider_mapping"] = currentProviderMappings
						}
					}
				}
			}

			// Interpolate variables in the scope
			scope, err := InterpolateScope(mapData.Scope, dataMap)

			if err != nil {
				log.WithContext(ctx).WithError(err).Debugf("Could not find scope mapping variables %v, adding them will result in better results. Error: ", mapData.Scope)
				scope = "*"
			}

			u := uuid.New()
			newQuery := &sdp.Query{
				Type:               mapData.Type,
				Method:             mapData.Method,
				Query:              fmt.Sprintf("%v", query),
				Scope:              scope,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
				Deadline:           timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			// cleanup item metadata from mapping query
			if itemDiff.Before != nil {
				itemDiff.Before.Type = newQuery.Type
				if newQuery.Scope != "*" {
					itemDiff.Before.Scope = newQuery.Scope
				}
			}

			// cleanup item metadata from mapping query
			if itemDiff.After != nil {
				itemDiff.After.Type = newQuery.Type
				if newQuery.Scope != "*" {
					itemDiff.After.Scope = newQuery.Scope
				}
			}

			plannedChangeGroups.Add(resourceChange.Type, &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: newQuery,
			})

			log.WithContext(ctx).WithFields(log.Fields{
				"scope":  newQuery.Scope,
				"type":   newQuery.Type,
				"query":  newQuery.Query,
				"method": newQuery.Method.String(),
			}).Debug("Mapped resource to query")
		}
	}

	supported := ""
	numSupported := plannedChangeGroups.NumSupportedChanges()
	if numSupported > 0 {
		supported = Green.Color(fmt.Sprintf("%v supported", numSupported))
	}

	unsupported := ""
	numUnsupported := plannedChangeGroups.NumUnsupportedChanges()
	if numUnsupported > 0 {
		unsupported = Yellow.Color(fmt.Sprintf("%v unsupported", numUnsupported))
	}

	numTotalChanges := numSupported + numUnsupported

	switch numTotalChanges {
	case 0:
		log.WithContext(ctx).Infof("Plan (%v) contained no changing resources.", fileName)
	case 1:
		log.WithContext(ctx).Infof("Plan (%v) contained one changing resource: %v %v", fileName, supported, unsupported)
	default:
		log.WithContext(ctx).Infof("Plan (%v) contained %v changing resources: %v %v", fileName, numTotalChanges, supported, unsupported)
	}

	// Log the types
	for typ, plannedChanges := range plannedChangeGroups.supported {
		log.WithContext(ctx).Infof(Green.Color("  ✓ %v (%v)"), typ, len(plannedChanges))
	}
	for typ, plannedChanges := range plannedChangeGroups.unsupported {
		log.WithContext(ctx).Infof(Yellow.Color("  ✗ %v (%v)"), typ, len(plannedChanges))
	}

	return plannedChangeGroups.MappedItemDiffs(), nil
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

func SubmitPlan(ctx context.Context, files []string, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI SubmitPlan", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	gatewayUrl := viper.GetString("gateway-url")
	if gatewayUrl == "" {
		gatewayUrl = fmt.Sprintf("%v/api/gateway", viper.GetString("url"))
		viper.Set("gateway-url", gatewayUrl)
	}

	lf := log.Fields{}

	ctx, err = ensureToken(ctx, []string{"changes:write"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithField("api-key-url", viper.GetString("api-key-url")).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fileWord := "file"
	if len(files) > 1 {
		fileWord = "files"
	}

	log.WithContext(ctx).Infof("Reading %v plan %v", len(files), fileWord)

	plannedChanges := make([]*sdp.MappedItemDiff, 0)

	for _, f := range files {
		lf["file"] = f
		mappedItemDiffs, err := mappedItemDiffsFromPlan(ctx, f, lf)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Error parsing terraform plan")
			return 1
		}
		plannedChanges = append(plannedChanges, mappedItemDiffs...)
	}
	delete(lf, "file")

	client := AuthenticatedChangesClient(ctx)
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, false)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed searching for existing changes")
		return 1
	}

	if changeUuid == uuid.Nil {
		title := changeTitle(viper.GetString("title"))
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  viper.GetString("ticket-link"),
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
				},
			},
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to create change")
			return 1
		}

		maybeChangeUuid := createResponse.Msg.Change.Metadata.GetUUIDParsed()
		if maybeChangeUuid == nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read change id")
			return 1
		}

		changeUuid = *maybeChangeUuid
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("Created a new change")
	} else {
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("Re-using change")
	}

	resultStream, err := client.UpdatePlannedChanges(ctx, &connect.Request[sdp.UpdatePlannedChangesRequest]{
		Msg: &sdp.UpdatePlannedChangesRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: plannedChanges,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to update planned changes")
		return 1
	}

	last_log := time.Now()
	first_log := true
	for resultStream.Receive() {
		msg := resultStream.Msg()

		// log the first message and at most every 250ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.State != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(lf).WithField("msg", msg).Info("Status update")
			last_log = time.Now()
			first_log = false
		}
	}
	if resultStream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(resultStream.Err()).Error("Error streaming results")
		return 1
	}

	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", viper.GetString("frontend"), changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("Change ready")
	fmt.Println(changeUrl)

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to get updated change")
		return 1
	}

	for _, a := range fetchResponse.Msg.Change.Properties.AffectedAppsUUID {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).WithField("app", a).Error("Received invalid app uuid")
			continue
		}
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"change-url": changeUrl,
			"app":        appUuid,
			"app-url":    fmt.Sprintf("%v/apps/%v", viper.GetString("frontend"), appUuid),
		}).Info("Affected app")
	}

	return 0
}

func init() {
	rootCmd.AddCommand(submitPlanCmd)

	submitPlanCmd.PersistentFlags().String("changes-url", "", "The changes service API endpoint (defaults to --url)")
	submitPlanCmd.PersistentFlags().String("management-url", "", "The management service API endpoint (defaults to --url)")
	submitPlanCmd.PersistentFlags().String("frontend", "https://app.overmind.tech", "The frontend base URL")

	submitPlanCmd.PersistentFlags().String("title", "", "Short title for this change. If this is not specified, ovm-cli will try to come up with one for you.")
	submitPlanCmd.PersistentFlags().String("description", "", "Quick description of the change.")
	submitPlanCmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change.")
	submitPlanCmd.PersistentFlags().String("owner", "", "The owner of this change.")
	// submitPlanCmd.PersistentFlags().String("cc-emails", "", "A comma-separated list of emails to keep updated with the status of this change.")

	submitPlanCmd.PersistentFlags().String("timeout", "3m", "How long to wait for responses")
}
