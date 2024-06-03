package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type snapshotModel struct {
	overall     taskModel
	discovering taskModel
	saving      taskModel

	state string
	items uint32
	edges uint32
}

type startSnapshotMsg struct {
	id int
}
type progressSnapshotMsg struct {
	id       int
	newState string
	items    uint32
	edges    uint32
}
type savingSnapshotMsg struct {
	id int
}
type finishSnapshotMsg struct {
	id int
}

func NewSnapShotModel(header, title string) snapshotModel {
	return snapshotModel{
		overall:     NewTaskModel(header),
		discovering: NewTaskModel(title),
		saving:      NewTaskModel("Saving"),
	}
}

func (m snapshotModel) Init() tea.Cmd {
	return tea.Batch(
		m.overall.Init(),
		m.discovering.Init(),
		m.saving.Init(),
	)
}

func (m snapshotModel) Update(msg tea.Msg) (snapshotModel, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case startSnapshotMsg:
		if m.overall.spinner.ID() != msg.id {
			return m, nil
		}
		m.overall.status = taskStatusRunning
		cmds = append(cmds, m.overall.spinner.Tick)
	case progressSnapshotMsg:
		if m.overall.spinner.ID() != msg.id {
			return m, nil
		}
		m.state = msg.newState
		m.items = msg.items
		m.edges = msg.edges

		m.discovering.status = taskStatusRunning
		cmds = append(cmds, m.discovering.spinner.Tick)
	case savingSnapshotMsg:
		if m.overall.spinner.ID() != msg.id {
			return m, nil
		}

		m.discovering.status = taskStatusDone

		m.saving.status = taskStatusRunning
		cmds = append(cmds, m.saving.spinner.Tick)

	case finishSnapshotMsg:
		if m.overall.spinner.ID() != msg.id {
			return m, nil
		}
		m.overall.status = taskStatusDone
		m.discovering.status = taskStatusDone
		m.saving.status = taskStatusDone
	default:
		var cmd tea.Cmd
		m.overall, cmd = m.overall.Update(msg)
		cmds = append(cmds, cmd)
		m.discovering, cmd = m.discovering.Update(msg)
		cmds = append(cmds, cmd)
		m.saving, cmd = m.saving.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m snapshotModel) View() string {
	// TODO: add progressbar; complication: we do not have a expected number of
	// items/edges to count towards for the progressbar

	// TODO: improve wrapping behaviour of the components. Currently skipped as
	// all the taskModel titles are expected to be relatively short and because
	// of the nesting of the components, the wrapping is more complex than the
	// current code structure supports
	bits := []string{}
	bits = append(bits, m.overall.View())

	itemStr := ""
	if m.items == 0 {
		itemStr = "0 items"
	} else if m.items == 1 {
		itemStr = "1 item"
	} else {
		itemStr = fmt.Sprintf("%d items", m.items)
	}

	edgeStr := ""
	if m.edges == 0 {
		edgeStr = "0 edges"
	} else if m.edges == 1 {
		edgeStr = "1 edge"
	} else {
		edgeStr = fmt.Sprintf("%d edges", m.edges)
	}

	detailStr := m.state
	if itemStr != "" || edgeStr != "" {
		detailStr = fmt.Sprintf("%s (%s, %s)", m.state, itemStr, edgeStr)
	}

	bits = append(bits, fmt.Sprintf("  %v - %v", m.discovering.View(), detailStr))
	bits = append(bits, fmt.Sprintf("  %v", m.saving.View()))
	return strings.Join(bits, "\n")
}

func (m snapshotModel) ID() int {
	return m.overall.spinner.ID()
}

func (m snapshotModel) StartMsg() tea.Msg {
	return startSnapshotMsg{
		id: m.overall.spinner.ID(),
	}
}

func (m snapshotModel) UpdateStatusMsg(newStatus taskStatus) tea.Msg {
	return m.overall.UpdateStatusMsg(newStatus)
}

func (m snapshotModel) ProgressMsg(newState string, items, edges uint32) tea.Msg {
	return progressSnapshotMsg{
		id:       m.overall.spinner.ID(),
		newState: newState,
		items:    items,
		edges:    edges,
	}
}
func (m snapshotModel) SavingMsg() tea.Msg {
	return savingSnapshotMsg{
		id: m.overall.spinner.ID(),
	}
}

func (m snapshotModel) FinishMsg() tea.Msg {
	return finishSnapshotMsg{
		id: m.overall.spinner.ID(),
	}
}
