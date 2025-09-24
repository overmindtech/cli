package tfutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestParseGCPProviders(t *testing.T) {
	tests := []struct {
		name           string
		terraformFile  string
		expectedCount  int
		expectedErrors int
	}{
		{
			name: "single google provider",
			terraformFile: `
provider "google" {
  project = "test-project"
  region  = "us-central1"
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "multiple google providers",
			terraformFile: `
provider "google" {
  project = "test-project-1"
  region  = "us-central1"
}

provider "google" {
  alias   = "west"
  project = "test-project-2"
  region  = "us-west1"
}`,
			expectedCount:  2,
			expectedErrors: 0,
		},
		{
			name: "google-beta provider",
			terraformFile: `
provider "google-beta" {
  project = "test-project"
  region  = "us-central1"
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "mixed providers with non-GCP",
			terraformFile: `
provider "aws" {
  region = "us-east-1"
}

provider "google" {
  project = "test-project"
  region  = "us-central1"
}

provider "azurerm" {
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.terraformFile), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse providers
			results, err := ParseGCPProviders(tmpDir, &hcl.EvalContext{}, false)
			if err != nil {
				t.Fatalf("ParseGCPProviders failed: %v", err)
			}

			// Count valid and error results
			validCount := 0
			errorCount := 0
			for _, result := range results {
				if result.Error != nil {
					errorCount++
				} else {
					validCount++
				}
			}

			if validCount != tt.expectedCount {
				t.Errorf("Expected %d valid providers, got %d", tt.expectedCount, validCount)
			}
			if errorCount != tt.expectedErrors {
				t.Errorf("Expected %d error providers, got %d", tt.expectedErrors, errorCount)
			}
		})
	}
}

func TestConfigFromGCPProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    GCPProvider
		expectError bool
		expectProj  string
		expectRegs  int
		expectZones int
	}{
		{
			name: "valid provider with region and zone",
			provider: GCPProvider{
				Project: "test-project",
				Region:  "us-central1",
				Zone:    "us-central1-a",
				Alias:   "test",
			},
			expectError: false,
			expectProj:  "test-project",
			expectRegs:  1,
			expectZones: 1,
		},
		{
			name: "valid provider with only project",
			provider: GCPProvider{
				Project: "test-project",
			},
			expectError: false,
			expectProj:  "test-project",
			expectRegs:  0,
			expectZones: 0,
		},
		{
			name: "missing project",
			provider: GCPProvider{
				Region: "us-central1",
			},
			expectError: true,
		},
		{
			name: "empty project",
			provider: GCPProvider{
				Project: "",
				Region:  "us-central1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ConfigFromGCPProvider(tt.provider)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				if config.ProjectID != tt.expectProj {
					t.Errorf("Expected project %s, got %s", tt.expectProj, config.ProjectID)
				}
				if len(config.Regions) != tt.expectRegs {
					t.Errorf("Expected %d regions, got %d", tt.expectRegs, len(config.Regions))
				}
				if len(config.Zones) != tt.expectZones {
					t.Errorf("Expected %d zones, got %d", tt.expectZones, len(config.Zones))
				}
				if config.Alias != tt.provider.Alias {
					t.Errorf("Expected alias %s, got %s", tt.provider.Alias, config.Alias)
				}
			}
		})
	}
}
