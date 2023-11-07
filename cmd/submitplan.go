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
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/cmd/datamaps"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/iter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

type MappedPlan struct {
	// Map of unsupported types and their changes
	UnsupportedChanges map[string][]ResourceChange

	// Map of supported types and their mapped queries
	SupportedChanges map[string][]TerraformToOvermindMapping
}

func (m MappedPlan) NumUnsupportedChanges() int {
	var num int

	for _, v := range m.UnsupportedChanges {
		num += len(v)
	}

	return num
}

func (m MappedPlan) NumSupportedChanges() int {
	var num int

	for _, v := range m.SupportedChanges {
		num += len(v)
	}

	return num
}

func (m MappedPlan) Queries() []*sdp.Query {
	queries := make([]*sdp.Query, 0)

	for _, mappings := range m.SupportedChanges {
		for _, mapping := range mappings {
			queries = append(queries, mapping.OvermindQuery)
		}
	}

	return queries
}

func NewMappedPlan() *MappedPlan {
	return &MappedPlan{
		UnsupportedChanges: make(map[string][]ResourceChange),
		SupportedChanges:   make(map[string][]TerraformToOvermindMapping),
	}
}

// Merges another mapped plan into this one
func (m *MappedPlan) Merge(other *MappedPlan) {
	for k, v := range other.UnsupportedChanges {
		m.UnsupportedChanges[k] = append(m.UnsupportedChanges[k], v...)
	}

	for k, v := range other.SupportedChanges {
		m.SupportedChanges[k] = append(m.SupportedChanges[k], v...)
	}
}

type TerraformToOvermindMapping struct {
	TerraformResource *Resource
	OvermindQuery     *sdp.Query
}

func changingItemQueriesFromPlan(ctx context.Context, fileName string, lf log.Fields) (*MappedPlan, error) {
	mappedPlan := NewMappedPlan()

	var overmindMappings []TerraformToOvermindMapping

	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return nil, err
	}

	var plan Plan
	err = json.Unmarshal(planJSON, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %v: %w", fileName, err)
	}

	// for all managed resources:
	for _, resourceChange := range plan.ResourceChanges {
		if len(resourceChange.Change.Actions) == 0 || resourceChange.Change.Actions[0] == "no-op" {
			// skip resources with no changes
			continue
		}

		// Track this change in the unsupported changes map. It will be moved to
		// supported later if we find a mapping
		mappedPlan.UnsupportedChanges[resourceChange.Type] = append(mappedPlan.UnsupportedChanges[resourceChange.Type], resourceChange)

		awsMappings := datamaps.AwssourceData[resourceChange.Type]
		k8sMappings := datamaps.K8ssourceData[resourceChange.Type]

		mappings := append(awsMappings, k8sMappings...)

		if len(mappings) == 0 {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
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
			dataMap := make(map[string]interface{})

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
			newQuery := sdp.Query{
				Type:               mapData.Type,
				Method:             mapData.Method,
				Query:              fmt.Sprintf("%v", query),
				Scope:              scope,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
				Deadline:           timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			overmindMappings = append(overmindMappings, TerraformToOvermindMapping{
				TerraformResource: currentResource,
				OvermindQuery:     &newQuery,
			})

			log.WithContext(ctx).WithFields(log.Fields{
				"scope":  newQuery.Scope,
				"type":   newQuery.Type,
				"query":  newQuery.Query,
				"method": newQuery.Method.String(),
			}).Debug("Mapped terraform to query")
		}
	}

	// Group mapped items by type
	for _, mapping := range overmindMappings {
		mappedPlan.SupportedChanges[mapping.TerraformResource.Type] = append(mappedPlan.SupportedChanges[mapping.TerraformResource.Type], mapping)
		// Delete supported type from unsupported map
		delete(mappedPlan.UnsupportedChanges, mapping.TerraformResource.Type)
	}

	resourceWord := "resource"
	if len(overmindMappings) > 1 {
		resourceWord = "resources"
	}

	supported := ""

	if mappedPlan.NumSupportedChanges() > 0 {
		supported = Green.Color(fmt.Sprintf("%v supported", mappedPlan.NumSupportedChanges()))
	}

	unsupported := ""

	if mappedPlan.NumUnsupportedChanges() > 0 {
		unsupported = Yellow.Color(fmt.Sprintf("%v unsupported", mappedPlan.NumUnsupportedChanges()))
	}

	totalChanges := mappedPlan.NumSupportedChanges() + mappedPlan.NumUnsupportedChanges()

	log.WithContext(ctx).Infof("Plan (%v) contained %v changing %v: %v %v", fileName, totalChanges, resourceWord, supported, unsupported)

	// Log the types
	for typ, mappings := range mappedPlan.SupportedChanges {
		log.WithContext(ctx).Infof(Green.Color("  ✓ %v (%v)"), typ, len(mappings))
	}

	for typ, mappings := range mappedPlan.UnsupportedChanges {
		log.WithContext(ctx).Infof(Yellow.Color("  ✗ %v (%v)"), typ, len(mappings))
	}

	return mappedPlan, nil
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

	planMappings := NewMappedPlan()

	for _, f := range files {
		lf["file"] = f
		mappings, err := changingItemQueriesFromPlan(ctx, f, lf)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("Error parsing terraform plan")
			return 1
		}
		planMappings.Merge(mappings)
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

	queries := planMappings.Queries()
	receivedItems := make([]*sdp.Item, 0)
	if len(queries) > 0 {
		mgmtClient := AuthenticatedManagementClient(ctx)
		log.WithContext(ctx).WithFields(lf).Info("Waking up sources")
		_, err = mgmtClient.KeepaliveSources(ctx, &connect.Request[sdp.KeepaliveSourcesRequest]{
			Msg: &sdp.KeepaliveSourcesRequest{
				WaitForHealthy: true,
			},
		})
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to wake up sources")
			return 1
		}

		ws, err := sdpws.Dial(ctx, gatewayUrl, otelhttp.DefaultClient, nil)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to gateway")
			return 1
		}

		results, err := iter.MapErr(queries, func(q **sdp.Query) ([]*sdp.Item, error) {
			return ws.Query(ctx, *q)
		})

		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to query items")
			return 1
		}

		for _, items := range results {
			receivedItems = append(receivedItems, items...)
		}

		// Print a summary of the results so far. I would like for this to be
		// nicer and do things like tell you why it failed, but for now this
		// will have to do
		for tfType, mappings := range planMappings.SupportedChanges {
			log.WithContext(ctx).Infof("  %v", tfType)

			for _, mapping := range mappings {
				queryUUID := mapping.OvermindQuery.ParseUuid()

				// Check for items matching this query UUID
				found := false

				for _, item := range receivedItems {
					if item.Metadata.SourceQuery.ParseUuid() == queryUUID {
						found = true
					}
				}

				if found {
					log.WithContext(ctx).Infof(Green.Color("    ✓ %v (found)"), mapping.TerraformResource.Name)
				} else {
					log.WithContext(ctx).Infof(Red.Color("    ✗ %v (not found)"), mapping.TerraformResource.Name)

					log.WithFields(log.Fields{
						"type":   mapping.OvermindQuery.Type,
						"scope":  mapping.OvermindQuery.Scope,
						"query":  mapping.OvermindQuery.Query,
						"method": mapping.OvermindQuery.Method.String(),
					}).Error("      No responses received")
				}
			}
		}
	} else {
		log.WithContext(ctx).WithFields(lf).Info("No item queries mapped, skipping changing items")
	}

	if len(receivedItems) > 0 {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("Updating changing items on the change record")
	} else {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("Updating change record with no items")
	}

	changingItemRefs := make([]*sdp.Reference, len(receivedItems))

	for i, item := range receivedItems {
		changingItemRefs[i] = item.Reference()
	}

	resultStream, err := client.UpdateChangingItems(ctx, &connect.Request[sdp.UpdateChangingItemsRequest]{
		Msg: &sdp.UpdateChangingItemsRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: changingItemRefs,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to update changing items")
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
