package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_ShowsUsageWithoutOptions(t *testing.T) {
	// Capture stdout and stderr
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Execute the command with --help flag to simulate usage request
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()

	// Get the output
	output := buf.String()

	// Verify that usage information is present in the output
	usageIndicators := []string{
		"gcp-source",
		"This sources looks for GCP resources in your account",
		"Usage:",
		"Flags:",
	}

	for _, indicator := range usageIndicators {
		if !strings.Contains(output, indicator) {
			t.Errorf("Expected usage output to contain %q, but it didn't. Output: %s", indicator, output)
		}
	}

	// --help should not produce an error
	if err != nil {
		t.Errorf("Expected Execute() with --help to return nil, but got error: %v", err)
	}
}
