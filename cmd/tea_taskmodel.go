package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type taskStatus int

const (
	taskStatusPending taskStatus = 0
	taskStatusRunning taskStatus = 1
	taskStatusDone    taskStatus = 2
	taskStatusError   taskStatus = 3
	taskStatusSkipped taskStatus = 4
)

type taskModel struct {
	status  taskStatus
	title   string
	spinner spinner.Model

	width  int
	indent int
}

type WithTaskModel interface {
	TaskModel() taskModel
}

// assert that taskModel implements WithTaskModel
var _ WithTaskModel = (*taskModel)(nil)

type updateTaskTitleMsg struct {
	id    int
	title string
}

type updateTaskStatusMsg struct {
	id     int
	status taskStatus
}

func NewTaskModel(title string, width int) taskModel {
	return taskModel{
		status: taskStatusPending,
		title:  title,
		spinner: spinner.New(
			spinner.WithSpinner(PlatformSpinner()),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorPalette.BgMain)),
		),
		width:  width,
		indent: 2,
	}
}

func (m taskModel) Init() tea.Cmd {
	if m.status == taskStatusRunning {
		return m.spinner.Tick
	}
	return nil
}

func (m taskModel) TaskModel() taskModel {
	return m
}

func (m taskModel) Update(msg tea.Msg) (taskModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(MAX_TERMINAL_WIDTH, msg.Width)
		return m, nil

	case updateTaskTitleMsg:
		if m.spinner.ID() == msg.id {
			m.title = msg.title
		}
	case updateTaskStatusMsg:
		if m.spinner.ID() == msg.id {
			m.status = msg.status
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m taskModel) View() string {
	label := ""
	switch m.status {
	case taskStatusPending:
		label = lipgloss.NewStyle().Foreground(ColorPalette.LabelFaint).Render("+")
	case taskStatusRunning:
		label = m.spinner.View()
	case taskStatusDone:
		label = RenderOk()
	case taskStatusError:
		label = RenderErr()
	case taskStatusSkipped:
		label = lipgloss.NewStyle().Foreground(ColorPalette.LabelFaint).Render("-")
	default:
		label = lipgloss.NewStyle().Render("?")
	}

	return wrap(fmt.Sprintf("%v %v", label, m.title), m.width, m.indent)
}

func (m taskModel) UpdateTitleMsg(newTitle string) tea.Msg {
	return updateTaskTitleMsg{
		id:    m.spinner.ID(),
		title: newTitle,
	}
}

func (m taskModel) UpdateStatusMsg(newStatus taskStatus) tea.Msg {
	return updateTaskStatusMsg{
		id:     m.spinner.ID(),
		status: newStatus,
	}
}
