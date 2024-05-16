package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// waitForCancellation returns a tea.Cmd that will wait for SIGINT and SIGTERM and run the provided cancel on receipt.
func waitForCancellation(ctx context.Context, cancel context.CancelFunc) tea.Cmd {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	return func() tea.Msg {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
		return tea.Quit
	}
}

// fatalError is a wrapper for errors that should abort the running tea.Program.
type fatalError struct {
	id  int
	err error
}

// otherError is a wrapper for errors that should NOT abort the running tea.Program.
type otherError struct {
	id  int
	err error
}

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
}

type WithTaskModel interface {
	TaskModel() taskModel
}

// assert that taskModel implements WithTaskModel
var _ WithTaskModel = (*taskModel)(nil)

func NewTaskModel(title string) taskModel {
	return taskModel{
		status: taskStatusPending,
		title:  title,
		spinner: spinner.New(
			spinner.WithSpinner(DotsSpinner),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorPalette.BgMain)),
		),
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
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	default:
		if m.status == taskStatusRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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
		label = lipgloss.NewStyle().Foreground(ColorPalette.BgSuccess).Render("✔︎")
	case taskStatusError:
		label = lipgloss.NewStyle().Foreground(ColorPalette.BgDanger).Render("x")
	case taskStatusSkipped:
		label = lipgloss.NewStyle().Foreground(ColorPalette.LabelFaint).Render("-")
	default:
		label = lipgloss.NewStyle().Render("?")
	}

	return fmt.Sprintf("%v %v", label, m.title)
}
