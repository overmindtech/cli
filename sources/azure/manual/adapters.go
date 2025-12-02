package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
)

// Adapters returns a slice of discovery.Adapter instances for Azure Source.
// It initializes Azure clients if initAzureClients is true, and creates adapters for the specified subscription ID and regions.
// Otherwise, it uses nil clients, which is useful for enumerating adapters for documentation purposes.
func Adapters(ctx context.Context, subscriptionID string, regions []string, cred *azidentity.DefaultAzureCredential, initAzureClients bool) ([]discovery.Adapter, error) {
	var adapters []discovery.Adapter

	if initAzureClients {
		if cred == nil {
			return nil, fmt.Errorf("credentials are required when initAzureClients is true")
		}

		log.WithFields(log.Fields{
			"ovm.source.subscription_id": subscriptionID,
			"ovm.source.regions":         regions,
		}).Info("Initializing Azure clients and discovering resource groups")

		// Create resource groups client to discover all resource groups in the subscription
		rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource groups client: %w", err)
		}

		// Discover resource groups in the subscription
		resourceGroups := make([]string, 0)
		pager := rgClient.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list resource groups: %w", err)
			}

			for _, rg := range page.Value {
				if rg.Name != nil {
					resourceGroups = append(resourceGroups, *rg.Name)
				}
			}
		}

		log.WithFields(log.Fields{
			"ovm.source.subscription_id":   subscriptionID,
			"ovm.source.resource_group_count": len(resourceGroups),
		}).Info("Discovered resource groups")

		// Initialize Azure SDK clients
		vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual machines client: %w", err)
		}

		// Create adapters for each resource group
		for _, resourceGroup := range resourceGroups {
			// Add Compute Virtual Machine adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewComputeVirtualMachine(
					clients.NewVirtualMachinesClient(vmClient),
					subscriptionID,
					resourceGroup,
				)),
			)
		}

		log.WithFields(log.Fields{
			"ovm.source.subscription_id": subscriptionID,
			"ovm.source.adapter_count":   len(adapters),
		}).Info("Initialized Azure adapters")

	} else {
		// For metadata registration only - no actual clients needed
		// This is used to enumerate available adapter types for documentation
		// Create placeholder adapters with nil clients for metadata registration
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeVirtualMachine(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
		)

		_ = regions
	}

	return adapters, nil
}
