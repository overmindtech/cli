package integrationtests

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"
)

// Shared constants for integration tests
const (
	integrationTestResourceGroup = "overmind-integration-tests"
	integrationTestLocation      = "westus2"
)

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
		Location: ptr.To(location),
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"managed": ptr.To("true"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create resource group: %w", err)
	}

	log.Printf("Resource group %s created successfully in location %s", resourceGroupName, location)
	return nil
}
