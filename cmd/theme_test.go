package cmd

import (
	"testing"
)

func TestMarkdownToString(t *testing.T) {
	// TODO: change this test data to use something that actually gets rendered to ANSI sequences and capture the correct output.
	markdown := "This is a test markdown"
	expectedOutput := "This is a test markdown"
	got := markdownToString(markdown)
	if got != expectedOutput {
		t.Errorf("Expected %q, but got %q", expectedOutput, got)
	}
}
