package manual

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/overmindtech/cli/discovery"
	// The following imports will be needed when implementing adapter registration:
	// "github.com/overmindtech/cli/sources"
	// "github.com/overmindtech/cli/sources/azure/clients"
)

// Adapters returns a slice of discovery.Adapter instances for Azure Source.
// It initializes Azure clients if initAzureClients is true, and creates adapters for the specified subscription ID and regions.
// Otherwise, it uses nil clients, which is useful for enumerating adapters for documentation purposes.
// TODO: fix function signature to use subscriptionID instead of projectID and remove zones/tokenSource parameters in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
func Adapters(ctx context.Context, subscriptionID string, regions []string, zones []string, tokenSource *oauth2.TokenSource, initAzureClients bool) ([]discovery.Adapter, error) {
	// TODO: instantiate Azure clients using federated credentials in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials

	var adapters []discovery.Adapter

	if initAzureClients {
		// TODO: Initialize Azure SDK clients using federated credentials in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
		// Example:
		// cred, err := azidentity.NewDefaultAzureCredential(nil)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to create Azure credential: %w", err)
		// }
		// vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to create virtual machines client: %w", err)
		// }

		// TODO: Discover resource groups in the subscription (requires ENG-1830 for authentication)
		// For now, this is a placeholder showing the pattern:
		// resourceGroups := []string{"rg-example-1", "rg-example-2"} // Would be discovered via Azure Resource Manager API
		// for _, resourceGroup := range resourceGroups {
		// 	adapters = append(adapters,
		// 		sources.WrapperToAdapter(NewComputeVirtualMachine(
		// 			clients.NewVirtualMachinesClient(vmClient),
		// 			subscriptionID,
		// 			resourceGroup,
		// 		)),
		//  )
		// }
		_ = subscriptionID // Suppress unused parameter warning - will be used when ENG-1830 is implemented
	} else {
		// Example: Compute Virtual Machine adapter registration pattern
		// This shows how adapters will be registered once ENG-1830 implements client initialization.
		// The actual registration requires:
		// 1. Azure SDK client initialized with federated credentials (ENG-1830)
		// 2. Resource group discovery (can be done via Azure Resource Manager API)
		//
		// Example pattern (commented until ENG-1830):
		// import (
		// 	"github.com/overmindtech/cli/sources"
		// 	"github.com/overmindtech/cli/sources/azure/clients"
		// )
		// vmClient := clients.NewVirtualMachinesClient(armcomputeClient) // Requires ENG-1830 for armcomputeClient
		// adapters = append(adapters,
		// 	sources.WrapperToAdapter(NewComputeVirtualMachine(
		// 		vmClient,
		// 		subscriptionID,
		// 		"example-resource-group", // Would be discovered from subscription
		// 	)),
		// )
		_ = subscriptionID // Suppress unused parameter warning - will be used when ENG-1830 is implemented
	}

	return adapters, nil
}
