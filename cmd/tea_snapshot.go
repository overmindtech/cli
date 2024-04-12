package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type snapshotModel struct {
	title string
	state string
	items uint32
	edges uint32
}

type startSnapshotMsg struct {
	newState string
}
type progressSnapshotMsg struct {
	newState string
	items    uint32
	edges    uint32
}
type finishSnapshotMsg struct {
	newState string
	items    uint32
	edges    uint32
}

func (m snapshotModel) Init() tea.Cmd {
	return nil
}

func (m snapshotModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startSnapshotMsg:
		m.state = msg.newState
	case progressSnapshotMsg:
		m.state = msg.newState
		m.items = msg.items
		m.edges = msg.edges
	case finishSnapshotMsg:
		m.state = msg.newState
		m.items = msg.items
		m.edges = msg.edges
	}
	return m, nil
}

func (m snapshotModel) View() string {
	// TODO: add spinner and/or progressbar; complication: we do not have a
	// expected number of items/edges to count towards for the progressbar
	return fmt.Sprintf("%s: %d items, %d edges", m.state, m.items, m.edges)
}
