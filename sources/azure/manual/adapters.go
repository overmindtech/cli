package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

// Adapters returns a slice of discovery.Adapter instances for Azure Source.
// It initializes Azure clients if initAzureClients is true, and creates adapters for the specified subscription ID and regions.
// Otherwise, it uses nil clients, which is useful for enumerating adapters for documentation purposes.
func Adapters(ctx context.Context, subscriptionID string, regions []string, cred *azidentity.DefaultAzureCredential, initAzureClients bool, cache sdpcache.Cache) ([]discovery.Adapter, error) {
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

		// Build resource group scopes for multi-scope adapters
		resourceGroupScopes := make([]azureshared.ResourceGroupScope, 0, len(resourceGroups))
		for _, rg := range resourceGroups {
			resourceGroupScopes = append(resourceGroupScopes, azureshared.NewResourceGroupScope(subscriptionID, rg))
		}

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

		postgresqlFlexibleServersClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgresql flexible servers client: %w", err)
		}

		secretsClient, err := armkeyvault.NewSecretsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create secrets client: %w", err)
		}

		userAssignedIdentitiesClient, err := armmsi.NewUserAssignedIdentitiesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create user assigned identities client: %w", err)
		}

		roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create role assignments client: %w", err)
		}

		diskEncryptionSetsClient, err := armcompute.NewDiskEncryptionSetsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create disk encryption sets client: %w", err)
		}

		imagesClient, err := armcompute.NewImagesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create images client: %w", err)
		}
		virtualMachineRunCommandsClient, err := armcompute.NewVirtualMachineRunCommandsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual machine run commands client: %w", err)
		}

		virtualMachineExtensionsClient, err := armcompute.NewVirtualMachineExtensionsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual machine extensions client: %w", err)
		}

		proximityPlacementGroupsClient, err := armcompute.NewProximityPlacementGroupsClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create proximity placement groups client: %w", err)
		}

		zonesClient, err := armdns.NewZonesClient(subscriptionID, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zones client: %w", err)
		}

		// Multi-scope resource group adapters (one adapter per type handling all resource groups)
		if len(resourceGroupScopes) > 0 {
			adapters = append(adapters,
				sources.WrapperToAdapter(NewComputeVirtualMachine(
					clients.NewVirtualMachinesClient(vmClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewStorageAccount(
					clients.NewStorageAccountsClient(storageAccountsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewStorageBlobContainer(
					clients.NewBlobContainersClient(blobContainersClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewStorageFileShare(
					clients.NewFileSharesClient(fileSharesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewStorageQueues(
					clients.NewQueuesClient(queuesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewStorageTable(
					clients.NewTablesClient(tablesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkVirtualNetwork(
					clients.NewVirtualNetworksClient(virtualNetworksClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkNetworkInterface(
					clients.NewNetworkInterfacesClient(networkInterfacesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewSqlDatabase(
					clients.NewSqlDatabasesClient(sqlDatabasesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewDocumentDBDatabaseAccounts(
					clients.NewDocumentDBDatabaseAccountsClient(documentDBDatabaseAccountsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewKeyVaultVault(
					clients.NewVaultsClient(keyVaultsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewKeyVaultManagedHSM(
					clients.NewManagedHSMsClient(managedHSMsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewDBforPostgreSQLDatabase(
					clients.NewPostgreSQLDatabasesClient(postgreSQLDatabasesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkPublicIPAddress(
					clients.NewPublicIPAddressesClient(publicIPAddressesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkLoadBalancer(
					clients.NewLoadBalancersClient(loadBalancersClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkZone(
					clients.NewZonesClient(zonesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewBatchAccount(
					clients.NewBatchAccountsClient(batchAccountsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeVirtualMachineScaleSet(
					clients.NewVirtualMachineScaleSetsClient(virtualMachineScaleSetsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeAvailabilitySet(
					clients.NewAvailabilitySetsClient(availabilitySetsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeDisk(
					clients.NewDisksClient(disksClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkNetworkSecurityGroup(
					clients.NewNetworkSecurityGroupsClient(networkSecurityGroupsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkRouteTable(
					clients.NewRouteTablesClient(routeTablesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewNetworkApplicationGateway(
					clients.NewApplicationGatewaysClient(applicationGatewaysClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewSqlServer(
					clients.NewSqlServersClient(sqlServersClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewDBforPostgreSQLFlexibleServer(
					clients.NewPostgreSQLFlexibleServersClient(postgresqlFlexibleServersClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewKeyVaultSecret(
					clients.NewSecretsClient(secretsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewManagedIdentityUserAssignedIdentity(
					clients.NewUserAssignedIdentitiesClient(userAssignedIdentitiesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewAuthorizationRoleAssignment(
					clients.NewRoleAssignmentsClient(roleAssignmentsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeDiskEncryptionSet(
					clients.NewDiskEncryptionSetsClient(diskEncryptionSetsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeImage(
					clients.NewImagesClient(imagesClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeVirtualMachineRunCommand(
					clients.NewVirtualMachineRunCommandsClient(virtualMachineRunCommandsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeVirtualMachineExtension(
					clients.NewVirtualMachineExtensionsClient(virtualMachineExtensionsClient),
					resourceGroupScopes,
				), cache),
				sources.WrapperToAdapter(NewComputeProximityPlacementGroup(
					clients.NewProximityPlacementGroupsClient(proximityPlacementGroupsClient),
					resourceGroupScopes,
				), cache),
			)
		}

		log.WithFields(log.Fields{
			"ovm.source.subscription_id": subscriptionID,
			"ovm.source.adapter_count":   len(adapters),
		}).Info("Initialized Azure adapters")

	} else {
		// For metadata registration only - no actual clients needed
		// This is used to enumerate available adapter types for documentation
		// Create placeholder adapters with nil clients and one placeholder scope
		placeholderResourceGroupScopes := []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, "placeholder-resource-group")}
		noOpCache := sdpcache.NewNoOpCache()
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeVirtualMachine(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewStorageAccount(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewStorageBlobContainer(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewStorageFileShare(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewStorageQueues(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewStorageTable(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkVirtualNetwork(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkNetworkInterface(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewSqlDatabase(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewDocumentDBDatabaseAccounts(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewKeyVaultVault(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewKeyVaultManagedHSM(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewDBforPostgreSQLDatabase(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkPublicIPAddress(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkLoadBalancer(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkZone(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewBatchAccount(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeVirtualMachineScaleSet(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeAvailabilitySet(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeDisk(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkNetworkSecurityGroup(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkRouteTable(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewNetworkApplicationGateway(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewSqlServer(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewDBforPostgreSQLFlexibleServer(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewKeyVaultSecret(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewManagedIdentityUserAssignedIdentity(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewAuthorizationRoleAssignment(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeDiskEncryptionSet(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeImage(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeVirtualMachineRunCommand(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeVirtualMachineExtension(nil, placeholderResourceGroupScopes), noOpCache),
			sources.WrapperToAdapter(NewComputeProximityPlacementGroup(nil, placeholderResourceGroupScopes), noOpCache),
		)

		_ = regions
	}

	return adapters, nil
}
