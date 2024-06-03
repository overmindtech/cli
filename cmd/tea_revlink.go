package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/overmindtech/sdp-go"
	"github.com/spf13/viper"
)

type revlinkWarmupFinishedMsg struct{}

type revlinkWarmupModel struct {
	taskModel

	ctx context.Context // note that this ctx is not initialized on NewGetConfigModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi  OvermindInstance
	// token *oauth2.Token

	status        chan *sdp.RevlinkWarmupResponse
	currentStatus *sdp.RevlinkWarmupResponse

	watchdogChan   chan struct{}      // a watchdog channel to keep the watchdog running
	watchdogCancel context.CancelFunc // the cancel function that gets called if the watchdog detects a timeout
}

func NewRevlinkWarmupModel() revlinkWarmupModel {
	return revlinkWarmupModel{
		taskModel: NewTaskModel("Discover and link all resources"),
		status:    make(chan *sdp.RevlinkWarmupResponse, 3000),
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

func (m revlinkWarmupModel) Update(msg tea.Msg) (revlinkWarmupModel, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case loadSourcesConfigMsg:
		m.ctx = msg.ctx
		m.oi = msg.oi
	case sourcesInitialisedMsg:
		m.taskModel.status = taskStatusRunning
		// start the spinner
		cmds = append(cmds, m.taskModel.spinner.Tick)

		// setup the watchdog infrastructure
		ctx, cancel := context.WithCancel(m.ctx)
		m.watchdogCancel = cancel

		// kick off a revlink warmup
		cmds = append(cmds, m.revlinkWarmupCmd(ctx))
		// process status updates
		cmds = append(cmds, m.waitForStatusActivity)

	case *sdp.RevlinkWarmupResponse:
		m.currentStatus = msg

		switch m.taskModel.status { //nolint:exhaustive // we only care about running and done
		case taskStatusRunning, taskStatusDone:
			items := m.currentStatus.GetItems()
			edges := m.currentStatus.GetEdges()
			if items+edges > 0 {
				m.taskModel.title = fmt.Sprintf("Discover and link all resources: %v (%v items, %v edges)", m.currentStatus.GetStatus(), items, edges)
			} else {
				m.taskModel.title = fmt.Sprintf("Discover and link all resources: %v", m.currentStatus.GetStatus())
			}
		}

		// wait for the next status update
		cmds = append(cmds, m.waitForStatusActivity)

		// tickle the watchdog when we get a response
		if m.watchdogChan != nil {
			go func() {
				m.watchdogChan <- struct{}{}
			}()
		}

	case runPlanFinishedMsg:
		if m.taskModel.status != taskStatusDone {
			// start the watchdog once the plan is done
			m.watchdogChan = make(chan struct{}, 1)
			cmds = append(cmds, m.watchdogCmd())
		}

	case revlinkWarmupFinishedMsg:
		m.taskModel.status = taskStatusDone
		m.watchdogChan = nil
		if m.watchdogCancel != nil {
			m.watchdogCancel()
			m.watchdogCancel = nil
		}
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		cmds = append(cmds, taskCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m revlinkWarmupModel) View() string {
	return m.taskModel.View()
}

// A command that waits for the activity on the status channel.
func (m revlinkWarmupModel) waitForStatusActivity() tea.Msg {
	return <-m.status
}

func (m revlinkWarmupModel) revlinkWarmupCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if viper.GetString("ovm-test-fake") != "" {
			for i := 0; i < 10; i++ {
				m.status <- &sdp.RevlinkWarmupResponse{
					Status: "running (test mode)",
					Items:  int32(i * 10),
					Edges:  int32(i*10) + 1,
				}
				time.Sleep(250 * time.Millisecond)
			}
			return revlinkWarmupFinishedMsg{}
		}

		client := AuthenticatedManagementClient(ctx, m.oi)

		stream, err := client.RevlinkWarmup(ctx, &connect.Request[sdp.RevlinkWarmupRequest]{
			Msg: &sdp.RevlinkWarmupRequest{},
		})
		if err != nil {
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error starting RevlinkWarmup: %w", err)}
		}

		for stream.Receive() {
			m.status <- stream.Msg()
		}

		err = stream.Err()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			return fatalError{id: m.spinner.ID(), err: fmt.Errorf("error warming up revlink: %w", stream.Err())}
		}

		return revlinkWarmupFinishedMsg{}
	}
}

func (m revlinkWarmupModel) watchdogCmd() tea.Cmd {
	return func() tea.Msg {
		ticker := time.NewTimer(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				m.watchdogCancel()
				return nil
			case <-m.ctx.Done():
				m.watchdogCancel()
				return nil
			case <-m.watchdogChan:
				// extend the timeout everytime we get a message
				ticker.Reset(10 * time.Second)
			}
		}
	}
}
