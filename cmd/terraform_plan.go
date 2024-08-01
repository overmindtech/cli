package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// terraformPlanCmd represents the `terraform plan` command
var terraformPlanCmd = &cobra.Command{
	Use:           "plan [overmind options...] -- [terraform options...]",
	Short:         "Runs `terraform plan` and sends the results to Overmind to calculate a blast radius and risks.",
	PreRun:        PreRunSetup,
	SilenceErrors: true,
	Run:           CmdWrapper("plan", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfPlanModel),
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

func addTerraformBaseFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("reset-stored-config", false, "[deprecated: this is now autoconfigured from local terraform files] Set this to reset the sources config stored in Overmind and input fresh values.")
	cmd.PersistentFlags().String("aws-config", "", "[deprecated: this is now autoconfigured from local terraform files] The chosen AWS config method, best set through the initial wizard when running the CLI. Options: 'profile_input', 'aws_profile', 'defaults', 'managed'.")
	cmd.PersistentFlags().String("aws-profile", "", "[deprecated: this is now autoconfigured from local terraform files] Set this to the name of the AWS profile to use.")
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("reset-stored-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-config"))
	cobra.CheckErr(cmd.PersistentFlags().MarkHidden("aws-profile"))
	cmd.PersistentFlags().Bool("only-use-managed-sources", false, "Set this to skip local autoconfiguration and only use the managed sources as configured in Overmind.")
}

func init() {
	terraformCmd.AddCommand(terraformPlanCmd)

	addAPIFlags(terraformPlanCmd)
	addChangeUuidFlags(terraformPlanCmd)
	addTerraformBaseFlags(terraformPlanCmd)
}

const TEST_RISK = `In publishing and graphic design, Lorem ipsum (/ˌlɔː.rəm ˈɪp.səm/) is a placeholder text commonly used to demonstrate the visual form of a document or a typeface without relying on meaningful content. Lorem ipsum may be used as a placeholder before the final copy is available. It is also used to temporarily replace text in a process called greeking, which allows designers to consider the form of a webpage or publication, without the meaning of the text influencing the design.

Lorem ipsum is typically a corrupted version of De finibus bonorum et malorum, a 1st-century BC text by the Roman statesman and philosopher Cicero, with words altered, added, and removed to make it nonsensical and improper Latin. The first two words themselves are a truncation of dolorem ipsum ("pain itself").`
