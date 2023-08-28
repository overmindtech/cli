package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/ttacon/chalk"
)

var tty bool

func init() {
	// Detect if we're in a TTY or not
	tty = isatty.IsTerminal(os.Stdout.Fd())
}

var (
	// Styles
	Underline = TextStyle{chalk.Underline}
	Bold      = TextStyle{chalk.Bold}

	// Colors
	Black   = Color{chalk.Black}
	Red     = Color{chalk.Red}
	Green   = Color{chalk.Green}
	Yellow  = Color{chalk.Yellow}
	Blue    = Color{chalk.Blue}
	Magenta = Color{chalk.Magenta}
	Cyan    = Color{chalk.Cyan}
	White   = Color{chalk.White}
)

// A type that wraps chalk.TextStyle but adds detections for if we're in a TTY
type TextStyle struct {
	underlying chalk.TextStyle
}

func (t TextStyle) TextStyle(val string) string {
	if !tty {
		// Don't style if we're not in a TTY
		return val
	}

	return t.underlying.TextStyle(val)
}

// A type that wraps chalk.Color but adds detections for if we're in a TTY
type Color struct {
	underlying chalk.Color
}

func (c Color) Color(val string) string {
	if !tty {
		// Don't style if we're not in a TTY
		return val
	}

	return c.underlying.Color(val)
}
