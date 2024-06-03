package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/overmindtech/cli/cmd"
)

func main() {
	// work around lipgloss/termenv integration bug.
	// See https://github.com/charmbracelet/lipgloss/issues/73#issuecomment-1144921037
	lipgloss.SetHasDarkBackground(termenv.HasDarkBackground())

	cmd.Execute()
}
