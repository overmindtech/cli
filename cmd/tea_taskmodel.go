package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
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
			spinner.WithSpinner(spinner.Spinner{
				Frames: []string{"∙∙∙∙∙∙∙", "●∙∙∙∙∙∙", "∙●∙∙∙∙∙", "∙∙●∙∙∙∙", "∙∙∙●∙∙∙", "∙∙∙∙●∙∙", "∙∙∙∙∙●∙", "∙∙∙∙∙∙●"},
				FPS:    time.Second / 7, //nolint:gomnd
			}),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPalette.Light.BgMain))),
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
		label = m.spinner.Style.Render("pending:")
	case taskStatusRunning:
		label = m.spinner.View()
		// all other lables are 7 cells wide
		for ansi.PrintableRuneWidth(label) <= 7 {
			label += " "
		}
	case taskStatusDone:
		label = m.spinner.Style.Render("done:   ")
	case taskStatusError:
		label = m.spinner.Style.Render("errored:")
	case taskStatusSkipped:
		label = m.spinner.Style.Render("skipped:")
	default:
		label = m.spinner.Style.Render("unknown:")
	}

	return fmt.Sprintf("%v %v", label, m.title)
}
