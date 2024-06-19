package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// terraformApplyCmd represents the `terraform apply` command
var terraformApplyCmd = &cobra.Command{
	Use:   "apply [overmind options...] -- [terraform options...]",
	Short: "Runs `terraform apply` between two full system configuration snapshots for tracking. This will be automatically connected with the Change created by the `plan` command.",
	PreRun: PreRunSetup,
	Run: CmdWrapper("apply", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfApplyModel),
}

type tfApplyModel struct {
	ctx context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance

	args []string

	planFile    string
	needPlan    bool
	runPlanTask runPlanModel

	runPlanFinished       bool
	revlinkWarmupFinished bool

	submitPlanTask submitPlanModel

	afterRisks       bool // indicate that risks have been displayed
	needApproval     bool
	planApprovalForm *huh.Form // this is set to true when we need to ask for approval, but we haven't yet received the approval

	changeUuid             uuid.UUID
	isStarting             bool
	startingChange         chan tea.Msg
	startingChangeSnapshot snapshotModel
	runTfApply             bool
	isEnding               bool
	endingChange           chan tea.Msg
	endingChangeSnapshot   snapshotModel

	parent *cmdModel
	width  int
}

type askForApprovalMsg struct{}
type showRisksMsg struct{}
type risksShownMsg struct{}
type startStartingSnapshotMsg struct{}

type changeIdentifiedMsg struct {
	uuid uuid.UUID
}

type runTfApplyMsg struct{}
type tfApplyFinishedMsg struct{}

func NewTfApplyModel(args []string, parent *cmdModel, width int) tea.Model {
	hasPlanSet := false
	autoapprove := false
	planFile := "overmind.plan"
	if len(args) >= 1 {
		f, err := os.Stat(args[len(args)-1])
		if err == nil && !f.IsDir() {
			// the last argument is a file, check that the previous arg is not
			// one that would eat this as argument
			hasPlanSet = true
			if len(args) >= 2 {
				prev := args[len(args)-2]
				for _, a := range []string{"-backup", "--backup", "-state", "--state", "-state-out", "--state-out"} {
					if prev == a || strings.HasPrefix(prev, a+"=") {
						hasPlanSet = false
						break
					}
				}
			}
		}
		if hasPlanSet {
			planFile = args[len(args)-1]
			autoapprove = true
		}
	}

	planArgs := append([]string{"plan"}, planArgsFromApplyArgs(args)...)

	if !hasPlanSet {
		// if the user has not set a plan, we need to set a temporary file to
		// capture the output for all calculations and to run apply afterwards

		f, err := os.CreateTemp("", "overmind-plan")
		if err != nil {
			log.WithError(err).Fatal("failed to create temporary plan file")
		}

		planFile = f.Name()

		planArgs = append(planArgs, "-out", planFile)
		args = append(args, planFile)

		// auto
		for _, a := range args {
			if a == "-auto-approve" || a == "-auto-approve=true" || a == "-auto-approve=TRUE" || a == "--auto-approve" || a == "--auto-approve=true" || a == "--auto-approve=TRUE" {
				autoapprove = true
			}
			if a == "-auto-approve=false" || a == "-auto-approve=FALSE" || a == "--auto-approve=false" || a == "--auto-approve=FALSE" {
				autoapprove = false
			}
		}
	}

	args = append([]string{"apply"}, args...)

	return tfApplyModel{
		args: args,

		planFile:        planFile,
		needPlan:        !hasPlanSet,
		runPlanTask:     NewRunPlanModel(planArgs, planFile, parent, width),
		runPlanFinished: hasPlanSet,

		submitPlanTask: NewSubmitPlanModel(planFile, width),

		needApproval: !autoapprove,

		startingChange:         make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		startingChangeSnapshot: NewSnapShotModel("Starting Change", "Taking snapshot", width),
		endingChange:           make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		endingChangeSnapshot:   NewSnapShotModel("Ending Change", "Taking snapshot", width),

		parent: parent,
	}
}

func (m tfApplyModel) Init() tea.Cmd {
	return nil
}

func (m tfApplyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)

	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case sourcesInitialisedMsg:
		if m.needPlan {
			cmds = append(
				cmds,
				m.runPlanTask.Init(),
				m.submitPlanTask.Init(),
			)
		} else {
			cmds = append(cmds,
				func() tea.Msg { return revlinkWarmupFinishedMsg{} },
				func() tea.Msg { return runPlanFinishedMsg{} },
			)
		}

	case revlinkWarmupFinishedMsg:
		m.revlinkWarmupFinished = true
		if m.runPlanFinished {
			cmds = append(cmds, func() tea.Msg {
				if m.needPlan {
					return submitPlanNowMsg{}
				} else {
					return startStartingSnapshotMsg{}
				}
			})
		}
	case runPlanFinishedMsg:
		m.runPlanFinished = true
		if m.revlinkWarmupFinished {
			cmds = append(cmds, func() tea.Msg {
				if m.needPlan {
					return submitPlanNowMsg{}
				} else {
					return startStartingSnapshotMsg{}
				}
			})
		}
	case submitPlanNowMsg:
		cmds = append(cmds, func() tea.Msg { return hideStartupStatusMsg{} })

	case submitPlanFinishedMsg:
		cmds = append(cmds, func() tea.Msg { return showRisksMsg{} })

	case showRisksMsg:
		cmds = append(cmds,
			tea.Sequence(
				func() tea.Msg { return freezeViewMsg{} },
				tea.Exec(
					m.parent.NewInterstitialCommand(fmt.Sprintf("%v\n%v", m.View(), m.submitPlanTask.FinalReport())),
					func(err error) tea.Msg {
						if err != nil {
							return fatalError{err: fmt.Errorf("failed to show risks: %w", err)}
						}
						return risksShownMsg{}
					})))

	case risksShownMsg:
		m.afterRisks = true
		cmds = append(cmds, func() tea.Msg { return unfreezeViewMsg{} })
		if m.needApproval {
			cmds = append(cmds, func() tea.Msg { return askForApprovalMsg{} })
		} else {
			cmds = append(cmds, func() tea.Msg { return startStartingSnapshotMsg{} })
		}

	case askForApprovalMsg:
		input := huh.NewInput().
			Key("approval").
			Title("Do you want to perform these actions?").
			Description("Terraform will perform the actions described above.\nOnly 'yes' will be accepted to approve.")
		m.planApprovalForm = huh.NewForm(
			huh.NewGroup(input),
		)
		cmds = append(cmds, input.Focus())

		if viper.GetString("ovm-test-fake") != "" {
			cmds = append(cmds, tea.Sequence(
				func() tea.Msg {
					return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("yes")}
				},
				func() tea.Msg {
					time.Sleep(time.Second)
					return tea.KeyMsg{Type: tea.KeyEnter}
				},
			))
		}

	case startStartingSnapshotMsg:
		m.isStarting = true
		cmds = append(cmds,
			m.startingChangeSnapshot.Init(),
			m.startStartChangeCmd(),
			m.waitForStartingActivity,
		)

	case changeIdentifiedMsg:
		m.changeUuid = msg.uuid
		cmds = append(cmds, m.waitForStartingActivity)

	case startSnapshotMsg:
		if msg.id == m.startingChangeSnapshot.ID() {
			cmds = append(cmds, m.waitForStartingActivity)
		} else if msg.id == m.endingChangeSnapshot.ID() {
			cmds = append(cmds, m.waitForEndingActivity)
		}

	case progressSnapshotMsg:
		if msg.id == m.startingChangeSnapshot.ID() {
			cmds = append(cmds, m.waitForStartingActivity)
		} else if msg.id == m.endingChangeSnapshot.ID() {
			cmds = append(cmds, m.waitForEndingActivity)
		}

	case finishSnapshotMsg:
		if msg.id == m.startingChangeSnapshot.ID() {
			m.isStarting = false
			// defer the actual command to give the view a chance to show the header
			m.runTfApply = true
			cmds = append(cmds, func() tea.Msg { return runTfApplyMsg{} })
		} else if msg.id == m.endingChangeSnapshot.ID() {
			cmds = append(cmds, func() tea.Msg { return delayQuitMsg{} })
		}

	case runTfApplyMsg:
		c := exec.CommandContext(m.ctx, "terraform", m.args...) // nolint:gosec // this is a user-provided command, let them do their thing
		// remove go's default process cancel behaviour, so that terraform has a
		// chance to gracefully shutdown when ^C is pressed. Otherwise the
		// process would get killed immediately and leave locks lingering behind
		c.Cancel = func() error {
			return nil
		}

		// inject the profile, if configured
		if aws_profile := viper.GetString("aws-profile"); aws_profile != "" {
			c.Env = append(c.Env, fmt.Sprintf("AWS_PROFILE=%v", aws_profile))
		}

		_, span := tracing.Tracer().Start(m.ctx, "terraform apply") // nolint:spancheck // will be ended in the tea.Exec cleanup func

		if viper.GetString("ovm-test-fake") != "" {
			c = exec.CommandContext(m.ctx, "bash", "-c", "for i in $(seq 25); do echo fake terraform apply progress line $i of 25; sleep .1; done")
		}
		return m, tea.Sequence( // nolint:spancheck // will be ended in the tea.Exec cleanup func
			func() tea.Msg { return freezeViewMsg{} },
			tea.Exec(
				m.parent.NewExecCommand(c),
				func(err error) tea.Msg {
					defer span.End()

					if err != nil {
						return fatalError{err: fmt.Errorf("failed to run terraform apply: %w", err)}
					}

					return tfApplyFinishedMsg{}
				}))
	case tfApplyFinishedMsg:
		m.runTfApply = false
		m.isEnding = true
		cmds = append(cmds,
			func() tea.Msg { return unfreezeViewMsg{} },
			func() tea.Msg { return hideStartupStatusMsg{} },
			m.endingChangeSnapshot.Init(),
			m.startEndChangeCmd(),
			m.waitForEndingActivity,
		)
	}

	mdl, cmd := m.startingChangeSnapshot.Update(msg)
	cmds = append(cmds, cmd)
	m.startingChangeSnapshot = mdl

	mdl, cmd = m.endingChangeSnapshot.Update(msg)
	cmds = append(cmds, cmd)
	m.endingChangeSnapshot = mdl

	if m.needPlan {
		mdl, cmd := m.runPlanTask.Update(msg)
		cmds = append(cmds, cmd)
		m.runPlanTask = mdl.(runPlanModel)

		mdl, cmd = m.submitPlanTask.Update(msg)
		cmds = append(cmds, cmd)
		m.submitPlanTask = mdl.(submitPlanModel)
	}

	if m.planApprovalForm != nil {
		mdl, cmd := m.planApprovalForm.Update(msg)
		cmds = append(cmds, cmd)
		m.planApprovalForm = mdl.(*huh.Form)

		switch m.planApprovalForm.State {
		case huh.StateAborted:
			// rejected
			cmds = append(cmds, tea.Quit)
			m.planApprovalForm = nil
		case huh.StateNormal:
			// wait for results
		case huh.StateCompleted:
			if m.planApprovalForm.GetString("approval") == "yes" {
				// approved
				cmds = append(cmds, func() tea.Msg { return startStartingSnapshotMsg{} })
			} else {
				// rejected
				cmds = append(cmds, tea.Quit)
			}
			m.planApprovalForm = nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m tfApplyModel) View() string {
	bits := []string{}

	if !m.afterRisks {
		if m.runPlanTask.status != taskStatusPending {
			bits = append(bits, m.runPlanTask.View())
		}

		if m.submitPlanTask.Status() != taskStatusPending {
			bits = append(bits, m.submitPlanTask.View())
		}
	}

	if m.planApprovalForm != nil {
		bits = append(bits, m.planApprovalForm.View())
	}

	if (m.isStarting || m.runTfApply) && m.startingChangeSnapshot.overall.status != taskStatusPending {
		bits = append(bits, m.startingChangeSnapshot.View())
	}

	if m.runTfApply {
		bits = append(bits,
			wrap(fmt.Sprintf("%v Running 'terraform %v'",
				RenderOk(),
				strings.Join(m.args, " "),
			), m.width, 2))
	}

	if m.isEnding && m.endingChangeSnapshot.overall.status != taskStatusPending {
		bits = append(bits,
			wrap(fmt.Sprintf("%v Ran 'terraform %v'",
				RenderOk(),
				strings.Join(m.args, " "),
			), m.width, 2))
		bits = append(bits, m.endingChangeSnapshot.View())
	}

	return strings.Join(bits, "\n") + "\n"
}

func (m tfApplyModel) startStartChangeCmd() tea.Cmd {
	ctx := m.ctx
	oi := m.oi

	return func() tea.Msg {
		if viper.GetString("ovm-test-fake") != "" {
			m.startingChange <- changeIdentifiedMsg{uuid: uuid.New()}
			m.startingChange <- m.startingChangeSnapshot.StartMsg()
			time.Sleep(time.Second)

			for i := 0; i < 5; i++ {
				m.startingChange <- m.startingChangeSnapshot.ProgressMsg(fmt.Sprintf("progress %v", i), uint32(i), uint32(i))
				time.Sleep(time.Second)
			}
			return m.startingChangeSnapshot.FinishMsg()
		}

		var err error
		ticketLink := viper.GetString("ticket-link")
		if ticketLink == "" {
			ticketLink, err = getTicketLinkFromPlan(m.planFile)
			if err != nil {
				return fatalError{err: err}
			}
		}

		changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, true)
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to identify change: %w", err)}
		}

		m.startingChange <- changeIdentifiedMsg{uuid: changeUuid}
		m.startingChange <- m.startingChangeSnapshot.StartMsg()

		client := AuthenticatedChangesClient(ctx, oi)
		startStream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
			Msg: &sdp.StartChangeRequest{
				ChangeUUID: changeUuid[:],
			},
		})
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to start change: %w", err)}
		}

		var msg *sdp.StartChangeResponse
		for startStream.Receive() {
			msg = startStream.Msg()
			log.WithFields(log.Fields{
				"state": msg.GetState(),
				"items": msg.GetNumItems(),
				"edges": msg.GetNumEdges(),
			}).Trace("progress")
			stateLabel := "unknown"
			switch msg.GetState() {
			case sdp.StartChangeResponse_STATE_UNSPECIFIED:
				stateLabel = "unknown"
			case sdp.StartChangeResponse_STATE_TAKING_SNAPSHOT:
				stateLabel = "capturing current state"
			case sdp.StartChangeResponse_STATE_SAVING_SNAPSHOT:
				stateLabel = "saving state"
			case sdp.StartChangeResponse_STATE_DONE:
				stateLabel = "done"
			}
			m.startingChange <- m.startingChangeSnapshot.ProgressMsg(stateLabel, msg.GetNumItems(), msg.GetNumEdges())
		}
		if startStream.Err() != nil {
			return fatalError{err: fmt.Errorf("failed to process start change: %w", startStream.Err())}
		}

		return m.startingChangeSnapshot.FinishMsg()
	}
}

// A command that waits for the activity on the startingChange channel.
func (m tfApplyModel) waitForStartingActivity() tea.Msg {
	return <-m.startingChange
}

func (m tfApplyModel) startEndChangeCmd() tea.Cmd {
	ctx := m.ctx
	oi := m.oi
	changeUuid := m.changeUuid

	return func() tea.Msg {
		if viper.GetString("ovm-test-fake") != "" {
			m.endingChange <- m.endingChangeSnapshot.StartMsg()
			time.Sleep(time.Second)

			for i := 0; i < 5; i++ {
				m.endingChange <- m.endingChangeSnapshot.ProgressMsg(fmt.Sprintf("progress %v", i), uint32(i), uint32(i))
				time.Sleep(time.Second)
			}
			return m.endingChangeSnapshot.FinishMsg()
		}

		m.endingChange <- m.endingChangeSnapshot.StartMsg()

		client := AuthenticatedChangesClient(ctx, oi)
		endStream, err := client.EndChange(ctx, &connect.Request[sdp.EndChangeRequest]{
			Msg: &sdp.EndChangeRequest{
				ChangeUUID: changeUuid[:],
			},
		})
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to end change: %w", err)}
		}

		var msg *sdp.EndChangeResponse
		for endStream.Receive() {
			msg = endStream.Msg()
			log.WithFields(log.Fields{
				"state": msg.GetState(),
				"items": msg.GetNumItems(),
				"edges": msg.GetNumEdges(),
			}).Trace("progress")
			stateLabel := "unknown"
			switch msg.GetState() {
			case sdp.EndChangeResponse_STATE_UNSPECIFIED:
				stateLabel = "unknown"
			case sdp.EndChangeResponse_STATE_TAKING_SNAPSHOT:
				stateLabel = "capturing current state"
			case sdp.EndChangeResponse_STATE_SAVING_SNAPSHOT:
				stateLabel = "saving state"
			case sdp.EndChangeResponse_STATE_DONE:
				stateLabel = "done"
			}
			m.endingChange <- m.endingChangeSnapshot.ProgressMsg(stateLabel, msg.GetNumItems(), msg.GetNumEdges())
		}
		if endStream.Err() != nil {
			return fatalError{err: fmt.Errorf("failed to process end change: %w", endStream.Err())}
		}

		return m.endingChangeSnapshot.FinishMsg()
	}
}

// A command that waits for the activity on the endingChange channel.
func (m tfApplyModel) waitForEndingActivity() tea.Msg {
	return <-m.endingChange
}

func init() {
	terraformCmd.AddCommand(terraformApplyCmd)

	addAPIFlags(terraformApplyCmd)
	addChangeUuidFlags(terraformApplyCmd)
	addTerraformBaseFlags(terraformApplyCmd)
}
