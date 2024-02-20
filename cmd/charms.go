package cmd

import (
	"fmt"

	"github.com/charmbracelet/glamour"
)

// NewTermRenderer returns a glamour.TermRenderer with overmind defaults or panics
func NewTermRenderer() *glamour.TermRenderer {
	r, err := glamour.NewTermRenderer(
		// detect background color and pick either the default dark or light theme
		glamour.WithAutoStyle(),
	)
	if err != nil {
		panic(fmt.Errorf("failed to initialize terminal renderer: %w", err))
	}
	if r == nil {
		panic("initialized terminal renderer is nil")
	}

	return r
}
