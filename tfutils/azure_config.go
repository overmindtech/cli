package tfutils

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// AzureProvider represents an Azure provider block in terraform files
// Based on: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs#argument-reference
type AzureProvider struct {
	Name           string `hcl:"name,label" yaml:"name,omitempty"`
	Alias          string `hcl:"alias,optional" yaml:"alias,omitempty"`
	SubscriptionID string `hcl:"subscription_id,optional" yaml:"subscription_id,omitempty"`
	TenantID       string `hcl:"tenant_id,optional" yaml:"tenant_id,omitempty"`
	ClientID       string `hcl:"client_id,optional" yaml:"client_id,omitempty"`
	ClientSecret   string `hcl:"client_secret,optional" yaml:"client_secret,omitempty"`
	Environment    string `hcl:"environment,optional" yaml:"environment,omitempty"`

	// Throw any additional stuff into here so it doesn't fail
	// This includes the required 'features' block and other optional blocks
	Remain hcl.Body `hcl:",remain" yaml:"-"`
}

// AzureProviderResult holds the result of parsing an Azure provider
type AzureProviderResult struct {
	Provider *AzureProvider
	Error    error
	FilePath string
}

// ParseAzureProviders scans for .tf files and extracts Azure provider configurations
// (azurerm). When recursive is false, only the provided directory is scanned;
// when true, the directory is walked recursively while skipping dot-directories
// (e.g., .terraform).
func ParseAzureProviders(terraformDir string, evalContext *hcl.EvalContext, recursive bool) ([]AzureProviderResult, error) {
	files, err := FindTerraformFiles(terraformDir, recursive)
	if err != nil {
		return nil, err
	}

	parser := hclparse.NewParser()
	results := make([]AzureProviderResult, 0)

	// Iterate over the files
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			results = append(results, AzureProviderResult{
				Error:    fmt.Errorf("error reading terraform file: (%v) %w", file, err),
				FilePath: file,
			})
			continue
		}

		// Parse the HCL file
		parsedFile, diag := parser.ParseHCL(b, file)
		if diag.HasErrors() {
			results = append(results, AzureProviderResult{
				Error:    fmt.Errorf("error parsing terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		// First decode really minimally to find just the Azure providers
		basicFile := basicProviderFile{}
		diag = gohcl.DecodeBody(parsedFile.Body, evalContext, &basicFile)
		if diag.HasErrors() {
			results = append(results, AzureProviderResult{
				Error:    fmt.Errorf("error decoding terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		for _, genericProvider := range basicFile.Providers {
			switch genericProvider.Name {
			case "azurerm":
				azureProvider := AzureProvider{
					// Since this was already decoded we need to use it here
					Name: genericProvider.Name,
				}
				diag = gohcl.DecodeBody(genericProvider.Remain, evalContext, &azureProvider)
				if diag.HasErrors() {
					results = append(results, AzureProviderResult{
						Error:    fmt.Errorf("error decoding terraform file: (%v) %w", file, diag),
						FilePath: file,
					})
					continue
				} else {
					results = append(results, AzureProviderResult{
						Provider: &azureProvider,
						FilePath: file,
					})
				}
			}
		}
	}

	return results, nil
}

// AzureConfig holds configuration for Azure source
type AzureConfig struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	Alias          string // Store alias for engine naming
}

// ConfigFromAzureProvider creates an AzureConfig from an AzureProvider.
// If subscription_id is not set in the provider, it falls back to environment variables
// (ARM_SUBSCRIPTION_ID or AZURE_SUBSCRIPTION_ID), matching the behavior of the
// Azure Terraform provider.
func ConfigFromAzureProvider(provider AzureProvider) (*AzureConfig, error) {
	config := &AzureConfig{
		SubscriptionID: provider.SubscriptionID,
		TenantID:       provider.TenantID,
		ClientID:       provider.ClientID,
		Alias:          provider.Alias,
	}

	// Fall back to environment variables if subscription_id not set in provider
	// ARM_SUBSCRIPTION_ID is used by the Azure Terraform provider
	// AZURE_SUBSCRIPTION_ID is used by the Azure SDK
	if config.SubscriptionID == "" {
		config.SubscriptionID = os.Getenv("ARM_SUBSCRIPTION_ID")
	}
	if config.SubscriptionID == "" {
		config.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	}

	// Similarly for tenant_id and client_id
	if config.TenantID == "" {
		config.TenantID = os.Getenv("ARM_TENANT_ID")
	}
	if config.TenantID == "" {
		config.TenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if config.ClientID == "" {
		config.ClientID = os.Getenv("ARM_CLIENT_ID")
	}
	if config.ClientID == "" {
		config.ClientID = os.Getenv("AZURE_CLIENT_ID")
	}

	if config.SubscriptionID == "" {
		return nil, fmt.Errorf("Azure provider must specify subscription_id (or set ARM_SUBSCRIPTION_ID/AZURE_SUBSCRIPTION_ID environment variable)")
	}

	return config, nil
}
