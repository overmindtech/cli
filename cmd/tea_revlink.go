package cmd

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
)

type revlinkWarmupFinishedMsg struct{}

type revlinkWarmupModel struct {
	taskModel

	ctx context.Context // note that this ctx is not initialized on NewGetConfigModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance
	// token *oauth2.Token

	status        chan *sdp.RevlinkWarmupResponse
	currentStatus *sdp.RevlinkWarmupResponse
}

func NewRevlinkWarmupModel() tea.Model {
	return revlinkWarmupModel{
		taskModel: NewTaskModel("Discover and link all resources"),
		status:    make(chan *sdp.RevlinkWarmupResponse),
		currentStatus: &sdp.RevlinkWarmupResponse{
			Status: "pending",
			Items:  0,
			Edges:  0,
		},
	}
}

func (m revlinkWarmupModel) TaskModel() taskModel {
	return m.taskModel
}

func (m revlinkWarmupModel) Init() tea.Cmd {
	return m.taskModel.Init()
}

func (m revlinkWarmupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi
	case sourcesInitialisedMsg:
		m.taskModel.status = taskStatusRunning
		// start the spinner
		cmds = append(cmds, m.taskModel.spinner.Tick)
		// kick off a revlink warmup
		cmds = append(cmds, m.revlinkWarmupCmd)
		// process status updates
		cmds = append(cmds, m.waitForStatusActivity)
	case *sdp.RevlinkWarmupResponse:
		m.currentStatus = msg
		// wait for the next status update
		cmds = append(cmds, m.waitForStatusActivity)
	case revlinkWarmupFinishedMsg:
		m.taskModel.status = taskStatusDone
	case fatalError:
		if msg.id == m.spinner.ID() {
			m.taskModel.status = taskStatusError
		}
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		cmds = append(cmds, taskCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m revlinkWarmupModel) View() string {
	view := m.taskModel.View()
	switch m.taskModel.status { //nolint:exhaustive // we only care about running and done
	case taskStatusRunning, taskStatusDone:
		items := m.currentStatus.GetItems()
		edges := m.currentStatus.GetEdges()
		if items+edges > 0 {
			view += fmt.Sprintf(": %v (%v items, %v edges)", m.currentStatus.GetStatus(), items, edges)
		} else {
			view += fmt.Sprintf(": %v", m.currentStatus.GetStatus())
		}
	}

	return view
}

// A command that waits for the activity on the status channel.
func (m revlinkWarmupModel) waitForStatusActivity() tea.Msg {
	msg := <-m.status
	log.Debugf("waitForStatusActivity received %T: %+v", msg, msg)
	return msg
}

func (m revlinkWarmupModel) revlinkWarmupCmd() tea.Msg {
	ctx := m.ctx

	client := AuthenticatedManagementClient(ctx, m.oi)

	stream, err := client.RevlinkWarmup(ctx, &connect.Request[sdp.RevlinkWarmupRequest]{
		Msg: &sdp.RevlinkWarmupRequest{},
	})

	if err != nil {
		return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error warming up revlink: %w", err)}
	}

	for stream.Receive() {
		m.status <- stream.Msg()
	}

	return revlinkWarmupFinishedMsg{}
}
