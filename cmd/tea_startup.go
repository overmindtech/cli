package cmd

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type instanceLoadedMsg struct {
	instance OvermindInstance
}

type instanceLoaderModel struct {
	taskModel
	ctx context.Context
	app string
}

func NewInstanceLoaderModel(ctx context.Context, app string) tea.Model {
	result := instanceLoaderModel{
		taskModel: NewTaskModel("Connecting to Overmind"),
		ctx:       ctx,
		app:       app,
	}
	result.status = taskStatusRunning
	return result
}

func (m instanceLoaderModel) Init() tea.Cmd {
	return tea.Batch(
		m.taskModel.Init(),
		newOvermindInstanceCmd(m.ctx, m.app),
	)
}

func (m instanceLoaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case instanceLoadedMsg:
		m.status = taskStatusDone
		m.title = "Connected to Overmind"
		return m, nil
	default:
		var cmd tea.Cmd
		if m.status == taskStatusRunning {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		} else {
			// stop updating the spinner if the task is not running
			return m, nil
		}
	}
}

func newOvermindInstanceCmd(ctx context.Context, app string) tea.Cmd {
	return func() tea.Msg {
		instance, err := NewOvermindInstance(ctx, app)
		if err != nil {
			return fatalError{err: fmt.Errorf("failed to get instance data from app: %w", err)}
		}

		return instanceLoadedMsg{instance}
	}
}
