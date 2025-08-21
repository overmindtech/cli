package tfutils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// GCPProvider represents a GCP provider block in terraform files
// Based on: https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference
type GCPProvider struct {
	Name                         string            `hcl:"name,label" yaml:"name,omitempty"`
	Alias                        string            `hcl:"alias,optional" yaml:"alias,omitempty"`
	Credentials                  string            `hcl:"credentials,optional" yaml:"credentials,omitempty"`
	AccessToken                  string            `hcl:"access_token,optional" yaml:"access_token,omitempty"`
	ImpersonateServiceAccount    string            `hcl:"impersonate_service_account,optional" yaml:"impersonate_service_account,omitempty"`
	Project                      string            `hcl:"project,optional" yaml:"project,omitempty"`
	Region                       string            `hcl:"region,optional" yaml:"region,omitempty"`
	Zone                         string            `hcl:"zone,optional" yaml:"zone,omitempty"`
	BillingProject               string            `hcl:"billing_project,optional" yaml:"billing_project,omitempty"`
	UserProjectOverride          bool              `hcl:"user_project_override,optional" yaml:"user_project_override,omitempty"`
	RequestTimeout               string            `hcl:"request_timeout,optional" yaml:"request_timeout,omitempty"`
	RequestReason                string            `hcl:"request_reason,optional" yaml:"request_reason,omitempty"`
	Scopes                       []string          `hcl:"scopes,optional" yaml:"scopes,omitempty"`
	DefaultLabels                map[string]string `hcl:"default_labels,optional" yaml:"default_labels,omitempty"`
	AddTerraformAttributionLabel bool              `hcl:"add_terraform_attribution_label,optional" yaml:"add_terraform_attribution_label,omitempty"`

	// Throw any additional stuff into here so it doesn't fail
	Remain hcl.Body `hcl:",remain" yaml:"-"`
}

type GCPProviderResult struct {
	Provider *GCPProvider
	Error    error
	FilePath string
}

// ParseGCPProviders parses GCP provider config from all terraform files in the given directory,
// similar to ParseAWSProviders but for GCP providers (google and google-beta)
func ParseGCPProviders(terraformDir string, evalContext *hcl.EvalContext) ([]GCPProviderResult, error) {
	files, err := filepath.Glob(filepath.Join(terraformDir, "*.tf"))
	if err != nil {
		return nil, err
	}

	parser := hclparse.NewParser()
	results := make([]GCPProviderResult, 0)

	// Iterate over the files
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			results = append(results, GCPProviderResult{
				Error:    fmt.Errorf("error reading terraform file: (%v) %w", file, err),
				FilePath: file,
			})
			continue
		}

		// Parse the HCL file
		parsedFile, diag := parser.ParseHCL(b, file)
		if diag.HasErrors() {
			results = append(results, GCPProviderResult{
				Error:    fmt.Errorf("error parsing terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		// First decode really minimally to find just the GCP providers
		basicFile := basicProviderFile{}
		diag = gohcl.DecodeBody(parsedFile.Body, evalContext, &basicFile)
		if diag.HasErrors() {
			results = append(results, GCPProviderResult{
				Error:    fmt.Errorf("error decoding terraform file: (%v) %w", file, diag),
				FilePath: file,
			})
			continue
		}

		for _, genericProvider := range basicFile.Providers {
			switch genericProvider.Name {
			case "google", "google-beta":
				gcpProvider := GCPProvider{
					// Since this was already decoded we need to use it here
					Name: genericProvider.Name,
				}
				diag = gohcl.DecodeBody(genericProvider.Remain, evalContext, &gcpProvider)
				if diag.HasErrors() {
					results = append(results, GCPProviderResult{
						Error:    fmt.Errorf("error decoding terraform file: (%v) %w", file, diag),
						FilePath: file,
					})
					continue
				} else {
					results = append(results, GCPProviderResult{
						Provider: &gcpProvider,
						FilePath: file,
					})
				}
			}
		}
	}

	return results, nil
}

// GCPConfig holds configuration for GCP source
type GCPConfig struct {
	ProjectID string
	Regions   []string
	Zones     []string
	Alias     string // Store alias for engine naming
}

// ConfigFromGCPProvider creates a GCPConfig from a GCPProvider
func ConfigFromGCPProvider(provider GCPProvider) (*GCPConfig, error) {
	config := &GCPConfig{
		ProjectID: provider.Project,
		Regions:   []string{},
		Zones:     []string{},
		Alias:     provider.Alias,
	}

	if provider.Region != "" {
		config.Regions = append(config.Regions, provider.Region)
	}

	if provider.Zone != "" {
		config.Zones = append(config.Zones, provider.Zone)
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("GCP provider must specify a project")
	}

	return config, nil
}
