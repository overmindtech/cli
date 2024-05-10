package cmd

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/oauth2"
)

type revlinkWarmupModel struct {
	taskModel

	ctx   context.Context // note that this ctx is not initialized on NewGetConfigModel to instead get a modified context through the loadSourcesConfigMsg that has a timeout and cancelFunction configured
	oi    OvermindInstance
	token *oauth2.Token
}

func NewRevlinkWarmupModel() tea.Model {
	return revlinkWarmupModel{
		taskModel: NewTaskModel("Discover and link all available resources"),
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
	case sourcesInitialisedMsg:
		// kick off a revlink warmup
	default:
		var taskCmd tea.Cmd
		m.taskModel, taskCmd = m.taskModel.Update(msg)
		cmds = append(cmds, taskCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m revlinkWarmupModel) View() string {
	view := m.taskModel.View()

	return view
}
