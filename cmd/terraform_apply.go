package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// terraformApplyCmd represents the `terraform apply` command
var terraformApplyCmd = &cobra.Command{
	Use:   "apply [overmind options...] -- [terraform options...]",
	Short: "Runs `terraform apply` between two full system configuration snapshots for tracking. This will be automatically connected with the Change created by the `plan` command.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `terraform apply` flags")
		}
	},
	Run: CmdWrapper("apply", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfApplyModel),
}

type tfApplyModel struct {
	ctx context.Context // note that this ctx is not initialized on NewTfPlanModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance

	args             []string
	applyHeader      string
	processingHeader string

	changeUuid             uuid.UUID
	isStarting             bool
	startingChange         chan tea.Msg
	startingChangeSnapshot snapshotModel
	runTfApply             bool
	isEnding               bool
	endingChange           chan tea.Msg
	endingChangeSnapshot   snapshotModel
	progress               []string
}

type changeIdentifiedMsg struct {
	uuid uuid.UUID
}

type runTfApplyMsg struct{}
type tfApplyFinishedMsg struct{}

func NewTfApplyModel(args []string) tea.Model {
	args = append([]string{"apply"}, args...)
	// plan file needs to go last
	args = append(args, "overmind.plan")

	// // TODO: remove this test setup
	// args = append([]string{"plan"}, args...)
	// // -out needs to go last to override whatever the user specified on the command line
	// args = append(args, "-out", "overmind.plan")

	applyHeader := `# Applying Changes

Running ` + "`" + `terraform %v` + "`\n"
	applyHeader = fmt.Sprintf(applyHeader, strings.Join(args, " "))

	processingHeader := `# Applying Changes

Applying changes with ` + "`" + `terraform %v` + "`\n"
	processingHeader = fmt.Sprintf(processingHeader, strings.Join(args, " "))

	return tfApplyModel{
		args:             args,
		applyHeader:      applyHeader,
		processingHeader: processingHeader,

		startingChange:         make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		startingChangeSnapshot: snapshotModel{title: "Starting Change", state: "pending"},
		endingChange:           make(chan tea.Msg, 10), // provide a small buffer for sending updates, so we don't block the processing
		endingChangeSnapshot:   snapshotModel{title: "Ending Change", state: "pending"},
		progress:               []string{},
	}
}

func (m tfApplyModel) Init() tea.Cmd {
	return nil
}

func (m tfApplyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi

	case revlinkWarmupFinishedMsg:
		m.isStarting = true
		return m, tea.Batch(
			m.startingChangeSnapshot.Init(),
			m.startStartChangeCmd(),
			m.waitForStartingActivity,
		)

	case changeIdentifiedMsg:
		m.changeUuid = msg.uuid
		return m, nil

	case startSnapshotMsg:
		if m.isStarting {
			m.startingChangeSnapshot.Update(msg)
			return m, m.waitForStartingActivity
		} else if m.isEnding {
			m.endingChangeSnapshot.Update(msg)
			return m, m.waitForEndingActivity
		}

	case progressSnapshotMsg:
		if m.isStarting {
			m.startingChangeSnapshot.Update(msg)
			return m, m.waitForStartingActivity
		} else if m.isEnding {
			m.endingChangeSnapshot.Update(msg)
			return m, m.waitForEndingActivity
		}

	case finishSnapshotMsg:
		if m.isStarting {
			m.startingChangeSnapshot.Update(msg)
			m.isStarting = false
			// defer the actual command to give the view a chance to show the header
			m.runTfApply = true
			return m, func() tea.Msg { return runTfApplyMsg{} }
		} else if m.isEnding {
			m.endingChangeSnapshot.Update(msg)
			return m, tea.Quit
		}

	case runTfApplyMsg:
		c := exec.CommandContext(m.ctx, "terraform", m.args...) // nolint:gosec // this is a user-provided command, let them do their thing

		// inject the profile, if configured
		if aws_profile := viper.GetString("aws-profile"); aws_profile != "" {
			c.Env = append(c.Env, fmt.Sprintf("AWS_PROFILE=%v", aws_profile))
		}
		return m, tea.ExecProcess(
			c,
			func(err error) tea.Msg {
				if err != nil {
					return fatalError{err: fmt.Errorf("failed to run terraform apply: %w", err)}
				}

				return tfApplyFinishedMsg{}
			})
	case tfApplyFinishedMsg:
		m.isEnding = true
		return m, tea.Batch(
			m.endingChangeSnapshot.Init(),
			m.startEndChangeCmd(),
			m.waitForEndingActivity,
		)
	}

	return m, nil
}

func (m tfApplyModel) View() string {
	if m.isStarting || m.runTfApply || m.isEnding {
		return markdownToString(m.processingHeader) + "\n" +
			m.startingChangeSnapshot.View() + "\n" +
			m.endingChangeSnapshot.View() + "\n" +
			strings.Join(m.progress, "\n") + "\n"
	}

	return markdownToString(m.applyHeader) + "\n"
}

func (m tfApplyModel) startStartChangeCmd() tea.Cmd {
	ctx := m.ctx
	oi := m.oi

	return func() tea.Msg {
		var err error
		ticketLink := viper.GetString("ticket-link")
		if ticketLink == "" {
			ticketLink, err = getTicketLinkFromPlan()
			if err != nil {
				return fatalError{err: err}
			}
		}

		changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, true)
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to identify change: %w", err)}
		}

		m.startingChange <- changeIdentifiedMsg{uuid: changeUuid}
		m.startingChange <- startSnapshotMsg{newState: "starting"}

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
			m.startingChange <- progressSnapshotMsg{
				newState: msg.GetState().String(),
				items:    msg.GetNumItems(),
				edges:    msg.GetNumEdges(),
			}
		}
		if startStream.Err() != nil {
			return fatalError{err: fmt.Errorf("failed to process start change: %w", startStream.Err())}
		}

		return finishSnapshotMsg{
			newState: msg.GetState().String(),
			items:    msg.GetNumItems(),
			edges:    msg.GetNumEdges(),
		}
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
		m.endingChange <- startSnapshotMsg{newState: "starting"}

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
			m.endingChange <- progressSnapshotMsg{
				newState: msg.GetState().String(),
				items:    msg.GetNumItems(),
				edges:    msg.GetNumEdges(),
			}
		}
		if endStream.Err() != nil {
			return fatalError{err: fmt.Errorf("failed to process end change: %w", endStream.Err())}
		}

		return finishSnapshotMsg{
			newState: msg.GetState().String(),
			items:    msg.GetNumItems(),
			edges:    msg.GetNumEdges(),
		}
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
