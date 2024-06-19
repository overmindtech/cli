package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/cmd/datamaps"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// terraformPlanCmd represents the `terraform plan` command
var terraformPlanCmd = &cobra.Command{
	Use:    "plan [overmind options...] -- [terraform options...]",
	Short:  "Runs `terraform plan` and sends the results to Overmind to calculate a blast radius and risks.",
	PreRun: PreRunSetup,
	Run:    CmdWrapper("plan", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfPlanModel),
}

type tfPlanModel struct {
	ctx context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance

	args        []string
	planFile    string
	runPlanTask runPlanModel

	runPlanFinished       bool
	revlinkWarmupFinished bool

	submitPlanTask submitPlanModel

	width int
}

// assert interface
var _ FinalReportingModel = (*tfPlanModel)(nil)

type mappedItemDiffsMsg struct {
	filename        string
	numSupported    int
	numUnsupported  int
	numTotalChanges int

	supported   map[string][]*sdp.MappedItemDiff
	unsupported map[string][]*sdp.MappedItemDiff
}

func NewTfPlanModel(args []string, parent *cmdModel, width int) tea.Model {
	hasPlanOutSet := false
	planFile := "overmind.plan"
	for i, a := range args {
		if a == "-out" || a == "--out=true" {
			hasPlanOutSet = true
			planFile = args[i+1]
		}
		if strings.HasPrefix(a, "-out=") {
			hasPlanOutSet = true
			planFile, _ = strings.CutPrefix(a, "-out=")
		}
		if strings.HasPrefix(a, "--out=") {
			hasPlanOutSet = true
			planFile, _ = strings.CutPrefix(a, "--out=")
		}
	}

	args = append([]string{"plan"}, args...)
	if !hasPlanOutSet {
		// if the user has not set a plan, we need to set a temporary file to
		// capture the output for the blast radius and risks calculation

		f, err := os.CreateTemp("", "overmind-plan")
		if err != nil {
			log.WithError(err).Fatal("failed to create temporary plan file")
		}

		planFile = f.Name()
		args = append(args, "-out", planFile)
		// TODO: remember whether we used a temporary plan file and remove it when done
	}

	return tfPlanModel{
		args:           args,
		runPlanTask:    NewRunPlanModel(args, planFile, parent, width),
		submitPlanTask: NewSubmitPlanModel(planFile, width),
		planFile:       planFile,
	}
}

func (m tfPlanModel) Init() tea.Cmd {
	return tea.Batch(
		m.runPlanTask.Init(),
		m.submitPlanTask.Init(),
	)
}

func (m tfPlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case revlinkWarmupFinishedMsg:
		m.revlinkWarmupFinished = true
		if m.runPlanFinished {
			cmds = append(cmds, func() tea.Msg { return submitPlanNowMsg{} })
		}
	case runPlanFinishedMsg:
		m.runPlanFinished = true
		cmds = append(cmds, func() tea.Msg { return hideStartupStatusMsg{} })
		if m.revlinkWarmupFinished {
			cmds = append(cmds, func() tea.Msg { return submitPlanNowMsg{} })
		}

	case submitPlanFinishedMsg:
		cmds = append(cmds, func() tea.Msg { return delayQuitMsg{} })
	}

	rpm, cmd := m.runPlanTask.Update(msg)
	m.runPlanTask = rpm.(runPlanModel)
	cmds = append(cmds, cmd)

	spm, cmd := m.submitPlanTask.Update(msg)
	m.submitPlanTask = spm.(submitPlanModel)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m tfPlanModel) View() string {
	bits := []string{}

	if m.runPlanTask.status != taskStatusPending {
		bits = append(bits, m.runPlanTask.View())
	}

	if m.submitPlanTask.Status() != taskStatusPending {
		bits = append(bits, m.submitPlanTask.View())
	}

	return strings.Join(bits, "\n") + "\n"
}

func (m tfPlanModel) FinalReport() string {
	return m.submitPlanTask.FinalReport()
}

// getTicketLinkFromPlan reads the plan file to create a unique hash to identify this change
func getTicketLinkFromPlan(planFile string) (string, error) {
	plan, err := os.ReadFile(planFile)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file (%v): %w", planFile, err)
	}
	h := sha256.New()
	h.Write(plan)
	return fmt.Sprintf("tfplan://{SHA256}%x", h.Sum(nil)), nil
}

func mappedItemDiffsFromPlanFile(ctx context.Context, fileName string, lf log.Fields) (int, []*sdp.MappedItemDiff, mappedItemDiffsMsg, error) {
	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return 0, nil, mappedItemDiffsMsg{}, err
	}

	return mappedItemDiffsFromPlan(ctx, planJSON, fileName, lf)
}

// mappedItemDiffsFromPlan takes a plan JSON, file name, and log fields as input
// and returns a slice of mapped item differences and an error. It parses the
// plan JSON, extracts resource changes, and creates mapped item differences for
// each resource change. It also generates mapping queries based on the resource
// type and current resource values. The function categorizes the mapped item
// differences into supported and unsupported changes. Finally, it logs the
// number of supported and unsupported changes and returns the mapped item
// differences.
func mappedItemDiffsFromPlan(ctx context.Context, planJson []byte, fileName string, lf log.Fields) (int, []*sdp.MappedItemDiff, mappedItemDiffsMsg, error) {
	// Check that we haven't been passed a state file
	if isStateFile(planJson) {
		return 0, nil, mappedItemDiffsMsg{}, fmt.Errorf("'%v' appears to be a state file, not a plan file", fileName)
	}

	var plan Plan
	err := json.Unmarshal(planJson, &plan)
	if err != nil {
		return 0, nil, mappedItemDiffsMsg{}, fmt.Errorf("failed to parse '%v': %w", fileName, err)
	}

	removedSecrets := countSensitiveValuesInConfig(plan.Config.RootModule) + countSensitiveValuesInState(plan.PlannedValues.RootModule)

	plannedChangeGroupsVar := plannedChangeGroups{
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
			return 0, nil, mappedItemDiffsMsg{}, fmt.Errorf("failed to create item diff for resource change: %w", err)
		}

		// Load mappings for this type. These mappings tell us how to create an
		// SDP query that will return this resource
		awsMappings := datamaps.AwssourceData[resourceChange.Type]
		k8sMappings := datamaps.K8ssourceData[resourceChange.Type]
		mappings := append(awsMappings, k8sMappings...)

		if len(mappings) == 0 {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
			plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
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
					WithField("terraform-query-field", mapData.QueryField).Warn("Adding unmapped resource")
				plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: nil, // unmapped item has no mapping query
				})
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
						// We need to split out the module section of the name
						// here. If the resource isn't in a module, the
						// ProviderConfigKey will be something like
						// "kubernetes", however if it's in a module it's be
						// something like "module.something:kubernetes"
						providerName := extractProviderNameFromConfigKey(configResource.ProviderConfigKey)
						currentProviderMappings, ok := mappings[providerName]

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
			if itemDiff.GetBefore() != nil {
				itemDiff.Before.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.Before.Scope = newQuery.GetScope()
				}
			}

			// cleanup item metadata from mapping query
			if itemDiff.GetAfter() != nil {
				itemDiff.After.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.After.Scope = newQuery.GetScope()
				}
			}

			plannedChangeGroupsVar.Add(resourceChange.Type, &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: newQuery,
			})

			log.WithContext(ctx).WithFields(log.Fields{
				"scope":  newQuery.GetScope(),
				"type":   newQuery.GetType(),
				"query":  newQuery.GetQuery(),
				"method": newQuery.GetMethod().String(),
			}).Debug("Mapped resource to query")
		}
	}

	supported := ""
	numSupported := plannedChangeGroupsVar.NumSupportedChanges()
	if numSupported > 0 {
		supported = Green.Color(fmt.Sprintf("%v supported", numSupported))
	}

	unsupported := ""
	numUnsupported := plannedChangeGroupsVar.NumUnsupportedChanges()
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
	for typ, plannedChanges := range plannedChangeGroupsVar.supported {
		log.WithContext(ctx).Infof("  %v %v (%v)", RenderOk(), typ, len(plannedChanges))
	}
	for typ, plannedChanges := range plannedChangeGroupsVar.unsupported {
		log.WithContext(ctx).Infof("  %v %v (%v)", RenderErr(), typ, len(plannedChanges))
	}

	return removedSecrets, plannedChangeGroupsVar.MappedItemDiffs(), mappedItemDiffsMsg{
		filename:        fileName,
		numSupported:    numSupported,
		numUnsupported:  numUnsupported,
		numTotalChanges: numTotalChanges,
		supported:       plannedChangeGroupsVar.supported,
		unsupported:     plannedChangeGroupsVar.unsupported,
	}, nil
}

func countSensitiveValuesInConfig(m ConfigModule) int {
	removedSecrets := 0
	for _, v := range m.Variables {
		if v.Sensitive {
			removedSecrets++
		}
	}
	for _, o := range m.Outputs {
		if o.Sensitive {
			removedSecrets++
		}
	}
	for _, c := range m.ModuleCalls {
		removedSecrets += countSensitiveValuesInConfig(c.Module)
	}
	return removedSecrets
}

func countSensitiveValuesInState(m Module) int {
	removedSecrets := 0
	for _, r := range m.Resources {
		removedSecrets += countSensitiveValuesInResource(r)
	}
	for _, c := range m.ChildModules {
		removedSecrets += countSensitiveValuesInState(c)
	}
	return removedSecrets
}

// follow itemAttributesFromResourceChangeData and maskSensitiveData
// implementation to count sensitive values
func countSensitiveValuesInResource(r Resource) int {
	// sensitiveMsg can be a bool or a map[string]any
	var isSensitive bool
	err := json.Unmarshal(r.SensitiveValues, &isSensitive)
	if err == nil && isSensitive {
		return 1 // one very large secret
	} else if err != nil {
		// only try parsing as map if parsing as bool failed
		var sensitive map[string]any
		err = json.Unmarshal(r.SensitiveValues, &sensitive)
		if err != nil {
			return 0
		}
		return countSensitiveAttributes(r.AttributeValues, sensitive)
	}
	return 0
}

func countSensitiveAttributes(attributes, sensitive any) int {
	if sensitive == true {
		return 1
	} else if sensitiveMap, ok := sensitive.(map[string]any); ok {
		if attributesMap, ok := attributes.(map[string]any); ok {
			result := 0
			for k, v := range attributesMap {
				result += countSensitiveAttributes(v, sensitiveMap[k])
			}
			return result
		} else {
			return 1
		}
	} else if sensitiveArr, ok := sensitive.([]any); ok {
		if attributesArr, ok := attributes.([]any); ok {
			if len(sensitiveArr) != len(attributesArr) {
				return 1
			}
			result := 0
			for i, v := range attributesArr {
				result += countSensitiveAttributes(v, sensitiveArr[i])
			}
			return result
		} else {
			return 1
		}
	}
	return 0
}

func addTerraformBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("reset-stored-config", false, "Set this to reset the sources config stored in Overmind and input fresh values.")
	cmd.PersistentFlags().String("aws-config", "", "The chosen AWS config method, best set through the initial wizard when running the CLI. Options: 'profile_input', 'aws_profile', 'defaults', 'managed'.")
	cmd.PersistentFlags().String("aws-profile", "", "Set this to the name of the AWS profile to use.")
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
	addChangeUuidFlags(terraformPlanCmd)
	addTerraformBaseFlags(terraformPlanCmd)
}

const TEST_RISK = `In publishing and graphic design, Lorem ipsum (/ˌlɔː.rəm ˈɪp.səm/) is a placeholder text commonly used to demonstrate the visual form of a document or a typeface without relying on meaningful content. Lorem ipsum may be used as a placeholder before the final copy is available. It is also used to temporarily replace text in a process called greeking, which allows designers to consider the form of a webpage or publication, without the meaning of the text influencing the design.

Lorem ipsum is typically a corrupted version of De finibus bonorum et malorum, a 1st-century BC text by the Roman statesman and philosopher Cicero, with words altered, added, and removed to make it nonsensical and improper Latin. The first two words themselves are a truncation of dolorem ipsum ("pain itself").`
