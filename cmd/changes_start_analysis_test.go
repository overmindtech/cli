package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestAddAnalysisFlags(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	addAnalysisFlags(cmd)

	tests := []struct {
		name     string
		flagName string
		flagType string
	}{
		{"blast-radius-link-depth", "blast-radius-link-depth", "int32"},
		{"blast-radius-max-items", "blast-radius-max-items", "int32"},
		{"blast-radius-max-time", "blast-radius-max-time", "duration"},
		{"change-analysis-target-duration", "change-analysis-target-duration", "duration"},
		{"signal-config", "signal-config", "string"},
		{"comment", "comment", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Expected flag %q to be registered", tt.flagName)
				return
			}
			if flag.Value.Type() != tt.flagType {
				t.Errorf("Expected flag %q to have type %q, got %q", tt.flagName, tt.flagType, flag.Value.Type())
			}
		})
	}

	// Verify blast-radius-max-time is deprecated
	flag := cmd.PersistentFlags().Lookup("blast-radius-max-time")
	if flag == nil {
		t.Error("Expected blast-radius-max-time flag to be registered")
		return
	}
	if flag.Deprecated == "" {
		t.Error("Expected blast-radius-max-time flag to be deprecated")
	}
}

func TestBuildAnalysisConfigWithNoFlags(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	ctx := context.Background()
	lf := log.Fields{}

	// When no flags are set, buildAnalysisConfig should succeed with nil/empty configs
	config, err := buildAnalysisConfig(ctx, lf)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	// BlastRadiusConfig should be nil when no flags are set
	if config.BlastRadiusConfig != nil {
		t.Errorf("Expected BlastRadiusConfig to be nil when no flags are set")
	}

	// RoutineChangesConfig should be nil when no signal config file exists
	if config.RoutineChangesConfig != nil {
		t.Errorf("Expected RoutineChangesConfig to be nil when no signal config exists")
	}

	// GithubOrgProfile should be nil when no signal config file exists
	if config.GithubOrgProfile != nil {
		t.Errorf("Expected GithubOrgProfile to be nil when no signal config exists")
	}
}

func TestBuildAnalysisConfigWithBlastRadiusFlags(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	viper.Set("blast-radius-link-depth", int32(5))
	viper.Set("blast-radius-max-items", int32(1000))

	ctx := context.Background()
	lf := log.Fields{}

	config, err := buildAnalysisConfig(ctx, lf)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.BlastRadiusConfig == nil {
		t.Fatal("Expected BlastRadiusConfig to be non-nil")
	}

	if config.BlastRadiusConfig.GetLinkDepth() != 5 {
		t.Errorf("Expected LinkDepth to be 5, got %d", config.BlastRadiusConfig.GetLinkDepth())
	}

	if config.BlastRadiusConfig.GetMaxItems() != 1000 {
		t.Errorf("Expected MaxItems to be 1000, got %d", config.BlastRadiusConfig.GetMaxItems())
	}
}

func TestBuildAnalysisConfigWithInvalidSignalConfigPath(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	// Set a non-existent signal config path
	viper.Set("signal-config", "/nonexistent/path/signal-config.yaml")

	ctx := context.Background()
	lf := log.Fields{}

	_, err := buildAnalysisConfig(ctx, lf)
	if err == nil {
		t.Fatal("Expected error for invalid signal config path")
	}
}

func TestBuildAnalysisConfigWithValidSignalConfig(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	// Create a temporary signal config file with valid content
	tempDir := t.TempDir()
	signalConfigPath := filepath.Join(tempDir, "signal-config.yaml")
	signalConfigContent := `routine_changes_config:
  sensitivity: 0
  duration_in_days: 1
  events_per_day: 1
`
	err := os.WriteFile(signalConfigPath, []byte(signalConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp signal config: %v", err)
	}

	viper.Set("signal-config", signalConfigPath)

	ctx := context.Background()
	lf := log.Fields{}

	config, err := buildAnalysisConfig(ctx, lf)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	// The signal config should be loaded
	if config.RoutineChangesConfig == nil {
		t.Error("Expected RoutineChangesConfig to be non-nil when signal config is loaded")
	}
}

func TestStartAnalysisCmdFlags(t *testing.T) {
	t.Parallel()

	// Verify the command has the expected flags registered
	tests := []struct {
		name     string
		flagName string
	}{
		{"wait flag", "wait"},
		{"ticket-link flag", "ticket-link"},
		{"uuid flag", "uuid"},
		{"change flag", "change"},
		{"app flag", "app"},
		{"timeout flag", "timeout"},
		// Analysis flags
		{"blast-radius-link-depth", "blast-radius-link-depth"},
		{"blast-radius-max-items", "blast-radius-max-items"},
		{"signal-config", "signal-config"},
		{"comment flag", "comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := startAnalysisCmd.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				// Check the parent command's flags
				flag = startAnalysisCmd.Flags().Lookup(tt.flagName)
			}
			if flag == nil {
				t.Errorf("Expected flag %q to be registered on start-analysis command", tt.flagName)
			}
		})
	}
}

func TestSubmitPlanCmdHasCommentFlag(t *testing.T) {
	t.Parallel()

	flag := submitPlanCmd.PersistentFlags().Lookup("comment")
	if flag == nil {
		t.Error("Expected comment flag to be registered on submit-plan command")
		return
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected comment flag default value to be 'false', got %q", flag.DefValue)
	}
}

func TestSubmitPlanCmdHasNoStartFlag(t *testing.T) {
	t.Parallel()

	flag := submitPlanCmd.PersistentFlags().Lookup("no-start")
	if flag == nil {
		t.Error("Expected no-start flag to be registered on submit-plan command")
		return
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected no-start flag default value to be 'false', got %q", flag.DefValue)
	}
}
