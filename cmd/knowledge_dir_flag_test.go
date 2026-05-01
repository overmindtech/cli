package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestKnowledgeDirFlagViperRoundTrip verifies that StringSlice + Viper correctly
// round-trips the --knowledge-dir flag value through both repeated and comma-separated formats.
// This is a defensive test against framework gotchas with StringSlice flag handling.
func TestKnowledgeDirFlagViperRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "empty flag",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "single directory",
			args:     []string{"--knowledge-dir", "/path/to/dir1"},
			expected: []string{"/path/to/dir1"},
		},
		{
			name:     "repeated flags",
			args:     []string{"--knowledge-dir", "/path/to/dir1", "--knowledge-dir", "/path/to/dir2"},
			expected: []string{"/path/to/dir1", "/path/to/dir2"},
		},
		{
			name:     "comma-separated",
			args:     []string{"--knowledge-dir", "/path/to/dir1,/path/to/dir2"},
			expected: []string{"/path/to/dir1", "/path/to/dir2"},
		},
		{
			name:     "mixed repeated and comma-separated",
			args:     []string{"--knowledge-dir", "/path/to/dir1", "--knowledge-dir", "/path/to/dir2,/path/to/dir3"},
			expected: []string{"/path/to/dir1", "/path/to/dir2", "/path/to/dir3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh viper instance for each test
			v := viper.New()

			// Create a test command with the knowledge-dir flag
			cmd := &cobra.Command{
				Use: "test",
				Run: func(cmd *cobra.Command, args []string) {},
			}
			cmd.Flags().StringSlice("knowledge-dir", []string{}, "Test flag")

			// Bind the flag to viper
			err := v.BindPFlag("knowledge-dir", cmd.Flags().Lookup("knowledge-dir"))
			if err != nil {
				t.Fatalf("failed to bind flag: %v", err)
			}

			// Parse the test args
			cmd.SetArgs(tt.args)
			err = cmd.Execute()
			if err != nil {
				t.Fatalf("failed to execute command: %v", err)
			}

			// Get the value from viper
			result := v.GetStringSlice("knowledge-dir")

			// Compare results
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d directories, got %d: expected=%v, got=%v", len(tt.expected), len(result), tt.expected, result)
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("directory at index %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}
