package cmd

import (
	"github.com/ttacon/chalk"
)

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

// A type that wraps chalk.Color but adds detections for if we're in a TTY
type Color struct {
	underlying chalk.Color
}
