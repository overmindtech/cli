package integrationtests

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
)

// Shared constants for integration tests
const (
	integrationTestResourceGroupBase = "overmind-integration-tests"
	integrationTestLocation          = "westus2"
)

var integrationTestResourceGroup = resolveIntegrationTestResourceGroup()

var invalidRunIDSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// resolveIntegrationTestResourceGroup returns the default integration test resource group,
// optionally scoped by AZURE_INTEGRATION_TEST_RUN_ID for parallel runs.
//
// Example:
//
//	AZURE_INTEGRATION_TEST_RUN_ID=agent-42
//	=> overmind-integration-tests-agent-42
func resolveIntegrationTestResourceGroup() string {
	runID := normalizeIntegrationTestRunID(os.Getenv("AZURE_INTEGRATION_TEST_RUN_ID"))
	if runID == "" {
		return integrationTestResourceGroupBase
	}

	// Azure resource group names can be up to 90 characters.
	name := integrationTestResourceGroupBase + "-" + runID
	if len(name) > 90 {
		return name[:90]
	}
	return name
}

func normalizeIntegrationTestRunID(runID string) string {
	normalized := strings.ToLower(strings.TrimSpace(runID))
	if normalized == "" {
		return ""
	}
	normalized = invalidRunIDSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if len(normalized) > 30 {
		normalized = normalized[:30]
	}
	return normalized
}

// createResourceGroup creates an Azure resource group if it doesn't already exist (idempotent)
func createResourceGroup(ctx context.Context, client *armresources.ResourceGroupsClient, resourceGroupName, location string) error {
	// Check if resource group already exists
	_, err := client.Get(ctx, resourceGroupName, nil)
	if err == nil {
		log.Printf("Resource group %s already exists, skipping creation", resourceGroupName)
		return nil
	}

	// Create the resource group
	_, err = client.CreateOrUpdate(ctx, resourceGroupName, armresources.ResourceGroup{
		Location: new(location),
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"managed": new("true"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create resource group: %w", err)
	}

	log.Printf("Resource group %s created successfully in location %s", resourceGroupName, location)
	return nil
}
