package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

type runPlanModel struct {
	ctx   context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi    OvermindInstance
	width int

	args     []string
	planFile string

	taskModel
}
type runPlanNowMsg struct{}
type runPlanFinishedMsg struct{}

func NewRunPlanModel(args []string, planFile string) runPlanModel {
	return runPlanModel{
		args:     args,
		planFile: planFile,

		taskModel: NewTaskModel("Planning Changes"),
	}
}

func (m runPlanModel) Init() tea.Cmd {
	return nil
}

func (m runPlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case sourcesInitialisedMsg:
		m.taskModel.status = taskStatusRunning
		// since the taskModel will not be shown while `terraform plan` is running,
		// there's no need to actually kick off the spinner
		// cmds = append(cmds, m.taskModel.spinner.Tick)

		// defer the actual command to give the view a chance to show the header
		cmds = append(cmds, func() tea.Msg { return runPlanNowMsg{} })

	case runPlanNowMsg:
		c := exec.CommandContext(m.ctx, "terraform", m.args...) // nolint:gosec // this is a user-provided command, let them do their thing

		// inject the profile, if configured
		if aws_profile := viper.GetString("aws-profile"); aws_profile != "" {
			c.Env = append(c.Env, fmt.Sprintf("AWS_PROFILE=%v", aws_profile))
		}

		if viper.GetString("ovm-test-fake") != "" {
			c = exec.CommandContext(m.ctx, "bash", "-c", "for i in $(seq 100); do echo fake terraform plan progress line $i of 100; done; sleep 1")
		}

		cmds = append(cmds, tea.ExecProcess(
			c,
			func(err error) tea.Msg {
				if err != nil {
					return fatalError{err: fmt.Errorf("failed to run terraform plan: %w", err)}
				}

				return runPlanFinishedMsg{}
			}))

	case runPlanFinishedMsg:
		m.taskModel.status = taskStatusDone

	default:
		// var cmd tea.Cmd
		// propagate commands to components
		// m.taskModel, cmd = m.taskModel.Update(msg)
		// cmds = append(cmds, cmd)

	}
	return m, tea.Batch(cmds...)
}

func (m runPlanModel) View() string {
	bits := []string{}

	switch m.taskModel.status {
	case taskStatusPending, taskStatusRunning:
		planHeader := `Running ` + "`" + `terraform %v` + "`\n"
		planHeader = fmt.Sprintf(planHeader, strings.Join(m.args, " "))
		bits = append(bits, markdownToString(planHeader))
	case taskStatusDone:
		bits = append(bits, m.taskModel.View())
	case taskStatusError, taskStatusSkipped:
		// handled by caller
	}

	if len(bits) > 0 {
		// ensure there's a newline at the end if and only if there's content
		bits = append(bits, "")
	}
	return strings.Join(bits, "\n")
}
