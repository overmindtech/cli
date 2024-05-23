package cmd

import (
	"fmt"
	"io"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type ExecCommandFunc func(cmd *exec.Cmd) tea.ExecCommand

// NewExecCommand returns a new ExecCommand that will print the last view from
// the parent cmdModel after bubbletea has released the terminal, but before the
// command is run.
func (m *cmdModel) NewExecCommand(c *exec.Cmd) tea.ExecCommand {
	return NewExecCommand(m, c)
}

func NewExecCommand(parent *cmdModel, c *exec.Cmd) *cliExecCommandModel {
	return &cliExecCommandModel{
		parent: parent,
		Cmd:    c,
	}
}

// osExecCommand is a layer over an exec.Cmd that satisfies the ExecCommand
// interface. It prints the last view from
// the parent cmdModel after bubbletea has released the terminal, but before the
// command is run.
type cliExecCommandModel struct {
	parent *cmdModel
	*exec.Cmd
}

func (c cliExecCommandModel) Run() error {
	_, err := c.Stdout.Write([]byte(c.parent.frozenView))
	if err != nil {
		return fmt.Errorf("failed to write view to stdout: %w", err)
	}
	return c.Cmd.Run()
}

// SetStdin sets stdin on underlying exec.Cmd to the given io.Reader.
func (c *cliExecCommandModel) SetStdin(r io.Reader) {
	// If unset, have the command use the same input as the terminal.
	if c.Stdin == nil {
		c.Stdin = r
	}
}

// SetStdout sets stdout on underlying exec.Cmd to the given io.Writer.
func (c *cliExecCommandModel) SetStdout(w io.Writer) {
	// If unset, have the command use the same output as the terminal.
	if c.Stdout == nil {
		c.Stdout = w
	}
}

// SetStderr sets stderr on the underlying exec.Cmd to the given io.Writer.
func (c *cliExecCommandModel) SetStderr(w io.Writer) {
	// If unset, use stderr for the command's stderr
	if c.Stderr == nil {
		c.Stderr = w
	}
}
