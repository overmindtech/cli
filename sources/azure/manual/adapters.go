package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
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
			"ovm.source.subscription_id":      subscriptionID,
			"ovm.source.resource_group_count": len(resourceGroups),
		}).Info("Discovered resource groups")

		// Initialize Azure SDK clients
		vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual machines client: %w", err)
		}

		storageAccountsClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage accounts client: %w", err)
		}

		blobContainersClient, err := armstorage.NewBlobContainersClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob containers client: %w", err)
		}

		fileSharesClient, err := armstorage.NewFileSharesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create file shares client: %w", err)
		}

		queuesClient, err := armstorage.NewQueueClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create queues client: %w", err)
		}

		tablesClient, err := armstorage.NewTableClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create tables client: %w", err)
		}

		virtualNetworksClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual networks client: %w", err)
		}

		networkInterfacesClient, err := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create network interfaces client: %w", err)
		}

		sqlDatabasesClient, err := armsql.NewDatabasesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create sql databases client: %w", err)
		}

		documentDBDatabaseAccountsClient, err := armcosmos.NewDatabaseAccountsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create document db database accounts client: %w", err)
		}

		keyVaultsClient, err := armkeyvault.NewVaultsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create key vaults client: %w", err)
		}

		postgreSQLDatabasesClient, err := armpostgresqlflexibleservers.NewDatabasesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgreSQL databases client: %w", err)
		}

		publicIPAddressesClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create public ip addresses client: %w", err)
		}

		loadBalancersClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancers client: %w", err)
		}

		batchAccountsClient, err := armbatch.NewAccountClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create batch accounts client: %w", err)
		}

		virtualMachineScaleSetsClient, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual machine scale sets client: %w", err)
		}

		availabilitySetsClient, err := armcompute.NewAvailabilitySetsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create availability sets client: %w", err)
		}

		disksClient, err := armcompute.NewDisksClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create disks client: %w", err)
		}
		networkSecurityGroupsClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create network security groups client: %w", err)
		}

		routeTablesClient, err := armnetwork.NewRouteTablesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create route tables client: %w", err)
		}

		applicationGatewaysClient, err := armnetwork.NewApplicationGatewaysClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create application gateways client: %w", err)
		}

		managedHSMsClient, err := armkeyvault.NewManagedHsmsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create managed hsms client: %w", err)
		}

		sqlServersClient, err := armsql.NewServersClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create sql servers client: %w", err)
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

			// Add Storage Account adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewStorageAccount(
					clients.NewStorageAccountsClient(storageAccountsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Storage Blob Container adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewStorageBlobContainer(
					clients.NewBlobContainersClient(blobContainersClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Storage File Share adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewStorageFileShare(
					clients.NewFileSharesClient(fileSharesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Storage Queue adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewStorageQueues(
					clients.NewQueuesClient(queuesClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Storage Table adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewStorageTable(
					clients.NewTablesClient(tablesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Network Virtual Network adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkVirtualNetwork(
					clients.NewVirtualNetworksClient(virtualNetworksClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Network Network Interface adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkNetworkInterface(
					clients.NewNetworkInterfacesClient(networkInterfacesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add SQL Database adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewSqlDatabase(
					clients.NewSqlDatabasesClient(sqlDatabasesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add DocumentDB Database Account adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewDocumentDBDatabaseAccounts(
					clients.NewDocumentDBDatabaseAccountsClient(documentDBDatabaseAccountsClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Key Vault Vault adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewKeyVaultVault(
					clients.NewVaultsClient(keyVaultsClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Key Vault Managed HSM adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewKeyVaultManagedHSM(
					clients.NewManagedHSMsClient(managedHSMsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add PostgreSQL Database adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewDBforPostgreSQLDatabase(
					clients.NewPostgreSQLDatabasesClient(postgreSQLDatabasesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Network Public IP Address adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkPublicIPAddress(
					clients.NewPublicIPAddressesClient(publicIPAddressesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Network Load Balancer adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkLoadBalancer(
					clients.NewLoadBalancersClient(loadBalancersClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Batch Account adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewBatchAccount(
					clients.NewBatchAccountsClient(batchAccountsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Virtual Machine Scale Set adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewComputeVirtualMachineScaleSet(
					clients.NewVirtualMachineScaleSetsClient(virtualMachineScaleSetsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Availability Set adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewComputeAvailabilitySet(
					clients.NewAvailabilitySetsClient(availabilitySetsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Disk adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewComputeDisk(
					clients.NewDisksClient(disksClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Network Security Group adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkNetworkSecurityGroup(
					clients.NewNetworkSecurityGroupsClient(networkSecurityGroupsClient),
					subscriptionID,
					resourceGroup,
				)),
			)
			// Add Network Route Table adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkRouteTable(
					clients.NewRouteTablesClient(routeTablesClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add Network Application Gateway adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewNetworkApplicationGateway(
					clients.NewApplicationGatewaysClient(applicationGatewaysClient),
					subscriptionID,
					resourceGroup,
				)),
			)

			// Add SQL Server adapter for this resource group
			adapters = append(adapters,
				sources.WrapperToAdapter(NewSqlServer(
					clients.NewSqlServersClient(sqlServersClient),
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
			sources.WrapperToAdapter(NewStorageAccount(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewStorageBlobContainer(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewStorageFileShare(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewStorageQueues(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewStorageTable(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkVirtualNetwork(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkNetworkInterface(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewSqlDatabase(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewDocumentDBDatabaseAccounts(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewKeyVaultVault(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewDBforPostgreSQLDatabase(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkPublicIPAddress(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkLoadBalancer(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewBatchAccount(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewComputeVirtualMachineScaleSet(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewComputeAvailabilitySet(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewComputeDisk(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkNetworkSecurityGroup(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkRouteTable(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewNetworkApplicationGateway(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewKeyVaultManagedHSM(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
			sources.WrapperToAdapter(NewSqlServer(
				nil, // nil client is okay for metadata registration
				subscriptionID,
				"placeholder-resource-group",
			)),
		)

		_ = regions
	}

	return adapters, nil
}
