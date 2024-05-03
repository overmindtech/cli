package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"
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
	log.Debugf("updated %+v => %v", msg, m)
	return m, nil
}

func (m snapshotModel) View() string {
	// TODO: add spinner and/or progressbar; complication: we do not have a
	// expected number of items/edges to count towards for the progressbar
	if m.items == 0 && m.edges == 0 {
		return fmt.Sprintf("%s - %s", m.title, m.state)
	} else if m.items == 1 && m.edges == 0 {
		return fmt.Sprintf("%s - %s: 1 item", m.title, m.state)
	} else if m.items == 1 && m.edges == 1 {
		return fmt.Sprintf("%s - %s: 1 item, 1 edge", m.title, m.state)
	} else if m.items > 1 && m.edges == 0 {
		return fmt.Sprintf("%s - %s: %d items", m.title, m.state, m.items)
	} else if m.items > 1 && m.edges == 1 {
		return fmt.Sprintf("%s - %s: %d items, 1 edge", m.title, m.state, m.items)
	} else {
		return fmt.Sprintf("%s - %s: %d items, %d edges", m.title, m.state, m.items, m.edges)
	}
}
