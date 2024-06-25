package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		cmds = append(cmds, func() tea.Msg { return hideStartupStatusMsg{} })
		if msg.err != nil {
			cmds = append(cmds, func() tea.Msg { return fatalError{err: msg.err} })
		} else {
			m.runPlanFinished = true
			if m.revlinkWarmupFinished {
				cmds = append(cmds, func() tea.Msg { return submitPlanNowMsg{} })
			}
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
