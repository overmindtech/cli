package custerm

import (
	"io"
	"strings"
	"time"

	"github.com/overmindtech/cli/custerm/internal"
	"github.com/pterm/pterm"
)

var activeSpinnerPrinters []*SpinnerPrinter

// DefaultSpinner is the default SpinnerPrinter.
var DefaultSpinner = SpinnerPrinter{
	Sequence:            []string{"▀ ", " ▀", " ▄", "▄ "},
	Style:               &pterm.ThemeDefault.SpinnerStyle,
	Delay:               time.Millisecond * 200,
	ShowTimer:           true,
	TimerRoundingFactor: time.Second,
	TimerStyle:          &pterm.ThemeDefault.TimerStyle,
	MessageStyle:        &pterm.ThemeDefault.SpinnerTextStyle,
	InfoPrinter:         &pterm.Info,
	SuccessPrinter:      &pterm.Success,
	FailPrinter:         &pterm.Error,
	WarningPrinter:      &pterm.Warning,
	Prefix: pterm.Prefix{
		Style: &pterm.ThemeDefault.SpinnerTextStyle,
		Text:  "",
	},
}

// SpinnerPrinter is a loading animation, which can be used if the progress is unknown.
// It's an animation loop, which can have a text and supports throwing errors or warnings.
// A TextPrinter is used to display all outputs, after the SpinnerPrinter is done.
type SpinnerPrinter struct {
	Text                string
	Sequence            []string
	Style               *pterm.Style
	Delay               time.Duration
	MessageStyle        *pterm.Style
	InfoPrinter         pterm.TextPrinter
	SuccessPrinter      pterm.TextPrinter
	FailPrinter         pterm.TextPrinter
	WarningPrinter      pterm.TextPrinter
	RemoveWhenDone      bool
	ShowTimer           bool
	TimerRoundingFactor time.Duration
	TimerStyle          *pterm.Style

	Prefix pterm.Prefix

	IsActive bool

	startedAt       time.Time
	currentSequence string

	Writer io.Writer
}

// WithText adds a text to the SpinnerPrinter.
func (s SpinnerPrinter) WithText(text string) *SpinnerPrinter {
	s.Text = text
	return &s
}

// WithSequence adds a sequence to the SpinnerPrinter.
func (s SpinnerPrinter) WithSequence(sequence ...string) *SpinnerPrinter {
	s.Sequence = sequence
	return &s
}

// WithStyle adds a style to the SpinnerPrinter.
func (s SpinnerPrinter) WithStyle(style *pterm.Style) *SpinnerPrinter {
	s.Style = style
	return &s
}

// WithDelay adds a delay to the SpinnerPrinter.
func (s SpinnerPrinter) WithDelay(delay time.Duration) *SpinnerPrinter {
	s.Delay = delay
	return &s
}

// WithMessageStyle adds a style to the SpinnerPrinter message.
func (s SpinnerPrinter) WithMessageStyle(style *pterm.Style) *SpinnerPrinter {
	s.MessageStyle = style
	return &s
}

// WithRemoveWhenDone removes the SpinnerPrinter after it is done.
func (s SpinnerPrinter) WithRemoveWhenDone(b ...bool) *SpinnerPrinter {
	s.RemoveWhenDone = internal.WithBoolean(b)
	return &s
}

// WithShowTimer shows how long the spinner is running.
func (s SpinnerPrinter) WithShowTimer(b ...bool) *SpinnerPrinter {
	s.ShowTimer = internal.WithBoolean(b)
	return &s
}

// WithTimerRoundingFactor sets the rounding factor for the timer.
func (s SpinnerPrinter) WithTimerRoundingFactor(factor time.Duration) *SpinnerPrinter {
	s.TimerRoundingFactor = factor
	return &s
}

// WithTimerStyle adds a style to the SpinnerPrinter timer.
func (s SpinnerPrinter) WithTimerStyle(style *pterm.Style) *SpinnerPrinter {
	s.TimerStyle = style
	return &s
}

// WithWriter sets the custom Writer.
func (p SpinnerPrinter) WithWriter(writer io.Writer) *SpinnerPrinter {
	p.Writer = writer
	return &p
}

// SetWriter sets the custom Writer.
func (p *SpinnerPrinter) SetWriter(writer io.Writer) {
	p.Writer = writer
}

// WithPrefix sets the prefix of the SpinnerPrinter.
func (s SpinnerPrinter) WithPrefix(prefix pterm.Prefix) *SpinnerPrinter {
	s.Prefix = prefix
	return &s
}

// WithIndentation sets the indentation of the SpinnerPrinter, without resetting
// the indentation's formatting.
func (s SpinnerPrinter) WithIndentation(indentation string) *SpinnerPrinter {
	s.Prefix.Text = indentation
	return &s
}

// GetFormattedPrefix returns the Prefix as a styled text string.
func (s SpinnerPrinter) GetFormattedPrefix() string {
	return s.Prefix.Style.Sprint(s.Prefix.Text)
}

// UpdateText updates the message of the active SpinnerPrinter.
// Can be used live.
func (s *SpinnerPrinter) UpdateText(text string) {
	s.Text = text
	if !pterm.RawOutput {
		pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.Style.Sprint(s.currentSequence)+" "+s.MessageStyle.Sprint(s.Text))
	} else {
		pterm.Fprintln(s.Writer, s.GetFormattedPrefix()+s.Text)
	}
}

// Start the SpinnerPrinter.
func (s SpinnerPrinter) Start(text ...interface{}) (*SpinnerPrinter, error) {
	s.IsActive = true
	s.startedAt = time.Now()
	activeSpinnerPrinters = append(activeSpinnerPrinters, &s)

	if len(text) != 0 {
		s.Text = pterm.Sprint(text...)
	}

	if pterm.RawOutput {
		pterm.Fprintln(s.Writer, s.Text)
	}

	go func() {
		for s.IsActive {
			for _, seq := range s.Sequence {
				if !s.IsActive {
					continue
				}
				if pterm.RawOutput {
					time.Sleep(s.Delay)
					continue
				}

				var timer string
				if s.ShowTimer {
					timer = " (" + time.Since(s.startedAt).Round(s.TimerRoundingFactor).String() + ")"
				}
				pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.Style.Sprint(seq)+" "+s.MessageStyle.Sprint(s.Text)+s.TimerStyle.Sprint(timer))
				s.currentSequence = seq
				time.Sleep(s.Delay)
			}
		}
	}()
	return &s, nil
}

// Stop terminates the SpinnerPrinter immediately.
// The SpinnerPrinter will not resolve into anything.
func (s *SpinnerPrinter) Stop() error {
	if !s.IsActive {
		return nil
	}
	s.IsActive = false
	if s.RemoveWhenDone {
		fClearLine(s.Writer)
		pterm.Fprinto(s.Writer)
	} else {
		pterm.Fprintln(s.Writer)
	}
	return nil
}

// GenericStart runs Start, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Start instead of this in your program.
func (s *SpinnerPrinter) GenericStart() (*pterm.LivePrinter, error) {
	p2, _ := s.Start()
	lp := pterm.LivePrinter(p2)
	return &lp, nil
}

// GenericStop runs Stop, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Stop instead of this in your program.
func (s *SpinnerPrinter) GenericStop() (*pterm.LivePrinter, error) {
	_ = s.Stop()
	lp := pterm.LivePrinter(s)
	return &lp, nil
}

// Info displays an info message
// If no message is given, the text of the SpinnerPrinter will be reused as the default message.
func (s *SpinnerPrinter) Info(message ...interface{}) {
	if s.InfoPrinter == nil {
		s.InfoPrinter = &pterm.Info
	}

	if len(message) == 0 {
		message = []interface{}{s.Text}
	}
	fClearLine(s.Writer)
	pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.InfoPrinter.Sprint(message...))
	_ = s.Stop()
}

// Success displays the success printer.
// If no message is given, the text of the SpinnerPrinter will be reused as the default message.
func (s *SpinnerPrinter) Success(message ...interface{}) {
	if s.SuccessPrinter == nil {
		s.SuccessPrinter = &pterm.Success
	}

	if len(message) == 0 {
		message = []interface{}{s.Text}
	}
	fClearLine(s.Writer)
	pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.SuccessPrinter.Sprint(message...))
	_ = s.Stop()
}

// Fail displays the fail printer.
// If no message is given, the text of the SpinnerPrinter will be reused as the default message.
func (s *SpinnerPrinter) Fail(message ...interface{}) {
	if s.FailPrinter == nil {
		s.FailPrinter = &pterm.Error
	}

	if len(message) == 0 {
		message = []interface{}{s.Text}
	}
	fClearLine(s.Writer)
	pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.FailPrinter.Sprint(message...))
	_ = s.Stop()
}

// Warning displays the warning printer.
// If no message is given, the text of the SpinnerPrinter will be reused as the default message.
func (s *SpinnerPrinter) Warning(message ...interface{}) {
	if s.WarningPrinter == nil {
		s.WarningPrinter = &pterm.Warning
	}

	if len(message) == 0 {
		message = []interface{}{s.Text}
	}
	fClearLine(s.Writer)
	pterm.Fprinto(s.Writer, s.GetFormattedPrefix()+s.WarningPrinter.Sprint(message...))
	_ = s.Stop()
}

func fClearLine(writer io.Writer) {
	pterm.Fprinto(writer, strings.Repeat(" ", pterm.GetTerminalWidth()))
}
