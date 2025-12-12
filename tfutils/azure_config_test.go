package tfutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestParseAzureProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		terraformFile  string
		expectedCount  int
		expectedErrors int
	}{
		{
			name: "single azurerm provider",
			terraformFile: `
provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  tenant_id       = "00000000-0000-0000-0000-000000000002"
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "multiple azurerm providers",
			terraformFile: `
provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  tenant_id       = "00000000-0000-0000-0000-000000000002"
  features {}
}

provider "azurerm" {
  alias           = "secondary"
  subscription_id = "00000000-0000-0000-0000-000000000003"
  tenant_id       = "00000000-0000-0000-0000-000000000004"
  features {}
}`,
			expectedCount:  2,
			expectedErrors: 0,
		},
		{
			name: "azurerm provider with client_id",
			terraformFile: `
provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  tenant_id       = "00000000-0000-0000-0000-000000000002"
  client_id       = "00000000-0000-0000-0000-000000000003"
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "mixed providers with non-Azure",
			terraformFile: `
provider "aws" {
  region = "us-east-1"
}

provider "google" {
  project = "test-project"
  region  = "us-central1"
}

provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "azurerm provider with environment",
			terraformFile: `
provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  environment     = "usgovernment"
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
		{
			name: "azurerm provider minimal config",
			terraformFile: `
provider "azurerm" {
  features {}
}`,
			expectedCount:  1,
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary directory and file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.tf")
			err := os.WriteFile(tmpFile, []byte(tt.terraformFile), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse providers
			results, err := ParseAzureProviders(tmpDir, &hcl.EvalContext{}, false)
			if err != nil {
				t.Fatalf("ParseAzureProviders failed: %v", err)
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

func TestParseAzureProvidersRecursive(t *testing.T) {
	t.Parallel()

	// Create temporary directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "submodule")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Main provider file
	mainTF := `
provider "azurerm" {
  subscription_id = "00000000-0000-0000-0000-000000000001"
  features {}
}`
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(mainTF), 0644)
	if err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Submodule provider file
	subTF := `
provider "azurerm" {
  alias           = "secondary"
  subscription_id = "00000000-0000-0000-0000-000000000002"
  features {}
}`
	err = os.WriteFile(filepath.Join(subDir, "providers.tf"), []byte(subTF), 0644)
	if err != nil {
		t.Fatalf("Failed to write submodule providers.tf: %v", err)
	}

	// Non-recursive should find only main
	results, err := ParseAzureProviders(tmpDir, &hcl.EvalContext{}, false)
	if err != nil {
		t.Fatalf("ParseAzureProviders (non-recursive) failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Non-recursive: expected 1 provider, got %d", len(results))
	}

	// Recursive should find both
	results, err = ParseAzureProviders(tmpDir, &hcl.EvalContext{}, true)
	if err != nil {
		t.Fatalf("ParseAzureProviders (recursive) failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Recursive: expected 2 providers, got %d", len(results))
	}
}

func TestConfigFromAzureProvider(t *testing.T) {
	// Note: These tests are not parallel because they modify environment variables

	t.Run("valid provider with all fields", func(t *testing.T) {
		provider := AzureProvider{
			SubscriptionID: "00000000-0000-0000-0000-000000000001",
			TenantID:       "00000000-0000-0000-0000-000000000002",
			ClientID:       "00000000-0000-0000-0000-000000000003",
			Alias:          "test",
		}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.SubscriptionID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("Expected subscription_id '00000000-0000-0000-0000-000000000001', got '%s'", config.SubscriptionID)
		}
		if config.TenantID != "00000000-0000-0000-0000-000000000002" {
			t.Errorf("Expected tenant_id '00000000-0000-0000-0000-000000000002', got '%s'", config.TenantID)
		}
		if config.ClientID != "00000000-0000-0000-0000-000000000003" {
			t.Errorf("Expected client_id '00000000-0000-0000-0000-000000000003', got '%s'", config.ClientID)
		}
		if config.Alias != "test" {
			t.Errorf("Expected alias 'test', got '%s'", config.Alias)
		}
	})

	t.Run("valid provider with only subscription_id", func(t *testing.T) {
		provider := AzureProvider{
			SubscriptionID: "00000000-0000-0000-0000-000000000001",
		}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.SubscriptionID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("Expected subscription_id '00000000-0000-0000-0000-000000000001', got '%s'", config.SubscriptionID)
		}
	})

	t.Run("missing subscription_id with no env vars", func(t *testing.T) {
		// Clear relevant env vars
		os.Unsetenv("ARM_SUBSCRIPTION_ID")
		os.Unsetenv("AZURE_SUBSCRIPTION_ID")

		provider := AzureProvider{
			TenantID: "00000000-0000-0000-0000-000000000002",
		}

		_, err := ConfigFromAzureProvider(provider)
		if err == nil {
			t.Error("Expected error but got none")
		}
	})

	t.Run("fallback to ARM_SUBSCRIPTION_ID env var", func(t *testing.T) {
		// Set ARM_SUBSCRIPTION_ID
		os.Setenv("ARM_SUBSCRIPTION_ID", "env-subscription-arm")
		defer os.Unsetenv("ARM_SUBSCRIPTION_ID")
		os.Unsetenv("AZURE_SUBSCRIPTION_ID")

		provider := AzureProvider{
			TenantID: "tenant-from-provider",
		}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.SubscriptionID != "env-subscription-arm" {
			t.Errorf("Expected subscription_id 'env-subscription-arm', got '%s'", config.SubscriptionID)
		}
	})

	t.Run("fallback to AZURE_SUBSCRIPTION_ID env var", func(t *testing.T) {
		// Set AZURE_SUBSCRIPTION_ID (ARM_ takes precedence, so unset it)
		os.Unsetenv("ARM_SUBSCRIPTION_ID")
		os.Setenv("AZURE_SUBSCRIPTION_ID", "env-subscription-azure")
		defer os.Unsetenv("AZURE_SUBSCRIPTION_ID")

		provider := AzureProvider{
			TenantID: "tenant-from-provider",
		}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.SubscriptionID != "env-subscription-azure" {
			t.Errorf("Expected subscription_id 'env-subscription-azure', got '%s'", config.SubscriptionID)
		}
	})

	t.Run("provider subscription_id takes precedence over env var", func(t *testing.T) {
		os.Setenv("ARM_SUBSCRIPTION_ID", "env-subscription")
		defer os.Unsetenv("ARM_SUBSCRIPTION_ID")

		provider := AzureProvider{
			SubscriptionID: "provider-subscription",
		}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.SubscriptionID != "provider-subscription" {
			t.Errorf("Expected subscription_id 'provider-subscription', got '%s'", config.SubscriptionID)
		}
	})

	t.Run("tenant_id and client_id fallback to env vars", func(t *testing.T) {
		os.Setenv("ARM_SUBSCRIPTION_ID", "sub")
		os.Setenv("ARM_TENANT_ID", "env-tenant")
		os.Setenv("ARM_CLIENT_ID", "env-client")
		defer func() {
			os.Unsetenv("ARM_SUBSCRIPTION_ID")
			os.Unsetenv("ARM_TENANT_ID")
			os.Unsetenv("ARM_CLIENT_ID")
		}()

		provider := AzureProvider{}

		config, err := ConfigFromAzureProvider(provider)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.TenantID != "env-tenant" {
			t.Errorf("Expected tenant_id 'env-tenant', got '%s'", config.TenantID)
		}
		if config.ClientID != "env-client" {
			t.Errorf("Expected client_id 'env-client', got '%s'", config.ClientID)
		}
	})
}

func TestParseAzureProviderValues(t *testing.T) {
	t.Parallel()

	terraformFile := `
provider "azurerm" {
  subscription_id = "sub-123"
  tenant_id       = "tenant-456"
  client_id       = "client-789"
  alias           = "primary"
  environment     = "public"
  features {}
}`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	err := os.WriteFile(tmpFile, []byte(terraformFile), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	results, err := ParseAzureProviders(tmpDir, &hcl.EvalContext{}, false)
	if err != nil {
		t.Fatalf("ParseAzureProviders failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Error != nil {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	provider := result.Provider
	if provider.SubscriptionID != "sub-123" {
		t.Errorf("Expected subscription_id 'sub-123', got '%s'", provider.SubscriptionID)
	}
	if provider.TenantID != "tenant-456" {
		t.Errorf("Expected tenant_id 'tenant-456', got '%s'", provider.TenantID)
	}
	if provider.ClientID != "client-789" {
		t.Errorf("Expected client_id 'client-789', got '%s'", provider.ClientID)
	}
	if provider.Alias != "primary" {
		t.Errorf("Expected alias 'primary', got '%s'", provider.Alias)
	}
	if provider.Environment != "public" {
		t.Errorf("Expected environment 'public', got '%s'", provider.Environment)
	}
	if provider.Name != "azurerm" {
		t.Errorf("Expected name 'azurerm', got '%s'", provider.Name)
	}
}
