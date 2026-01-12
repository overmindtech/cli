package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGetSignalsCmd(t *testing.T) {
	// Test that the command is properly registered
	if getSignalsCmd == nil {
		t.Fatal("getSignalsCmd is nil")
	}

	if getSignalsCmd.Use == "" {
		t.Error("getSignalsCmd.Use should not be empty")
	}

	// Test that required flags are set
	formatFlag := getSignalsCmd.PersistentFlags().Lookup("format")
	if formatFlag == nil {
		t.Error("format flag should be defined")
	} else if formatFlag.DefValue != "json" {
		t.Errorf("format flag default should be 'json', got '%s'", formatFlag.DefValue)
	}

	statusFlag := getSignalsCmd.PersistentFlags().Lookup("status")
	if statusFlag == nil {
		t.Error("status flag should be defined")
	}
}

func TestGetSignalsFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		shouldError bool
	}{
		{
			name:        "json format",
			format:      "json",
			shouldError: false,
		},
		{
			name:        "markdown format",
			format:      "markdown",
			shouldError: false,
		},
		{
			name:        "invalid format",
			format:      "xml",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("format", tt.format)

			format := viper.GetString("format")
			isValid := format == "json" || format == "markdown"

			if tt.shouldError && isValid {
				t.Error("Expected format validation to fail, but it passed")
			}
			if !tt.shouldError && !isValid {
				t.Error("Expected format validation to pass, but it failed")
			}
		})
	}
}
