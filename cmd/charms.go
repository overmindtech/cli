package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
)

var accessibleMode bool = os.Getenv("ACCESSIBLE") != ""

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
