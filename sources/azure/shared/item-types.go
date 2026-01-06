package shared

import "github.com/overmindtech/cli/sources/shared"

// Item types for Azure resources
// These combine the Azure source, API (resource provider), and resource type
// to create unique item type identifiers following the pattern: azure-{api}-{resource}
var (
	// Compute item types
	ComputeVirtualMachine           = shared.NewItemType(Azure, Compute, VirtualMachine)
	ComputeDisk                     = shared.NewItemType(Azure, Compute, Disk)
	ComputeAvailabilitySet          = shared.NewItemType(Azure, Compute, AvailabilitySet)
	ComputeVirtualMachineExtension  = shared.NewItemType(Azure, Compute, VirtualMachineExtension)
	ComputeVirtualMachineRunCommand = shared.NewItemType(Azure, Compute, VirtualMachineRunCommand)

	// Network item types
	NetworkVirtualNetwork                      = shared.NewItemType(Azure, Network, VirtualNetwork)
	NetworkSubnet                              = shared.NewItemType(Azure, Network, Subnet)
	NetworkNetworkInterface                    = shared.NewItemType(Azure, Network, NetworkInterface)
	NetworkPublicIPAddress                     = shared.NewItemType(Azure, Network, PublicIPAddress)
	NetworkNetworkSecurityGroup                = shared.NewItemType(Azure, Network, NetworkSecurityGroup)
	NetworkVirtualNetworkPeering               = shared.NewItemType(Azure, Network, VirtualNetworkPeering)
	NetworkNetworkInterfaceIPConfiguration     = shared.NewItemType(Azure, Network, NetworkInterfaceIPConfiguration)
	NetworkPrivateEndpoint                     = shared.NewItemType(Azure, Network, PrivateEndpoint)
	NetworkLoadBalancer                        = shared.NewItemType(Azure, Network, LoadBalancer)
	NetworkLoadBalancerFrontendIPConfiguration = shared.NewItemType(Azure, Network, LoadBalancerFrontendIPConfiguration)
	NetworkLoadBalancerBackendAddressPool      = shared.NewItemType(Azure, Network, LoadBalancerBackendAddressPool)
	NetworkLoadBalancerInboundNatRule          = shared.NewItemType(Azure, Network, LoadBalancerInboundNatRule)
	NetworkLoadBalancerLoadBalancingRule       = shared.NewItemType(Azure, Network, LoadBalancerLoadBalancingRule)
	NetworkLoadBalancerProbe                   = shared.NewItemType(Azure, Network, LoadBalancerProbe)
	NetworkLoadBalancerOutboundRule            = shared.NewItemType(Azure, Network, LoadBalancerOutboundRule)
	NetworkLoadBalancerInboundNatPool          = shared.NewItemType(Azure, Network, LoadBalancerInboundNatPool)
	NetworkPublicIPPrefix                      = shared.NewItemType(Azure, Network, PublicIPPrefix)
	NetworkNatGateway                          = shared.NewItemType(Azure, Network, NatGateway)
	NetworkDdosProtectionPlan                  = shared.NewItemType(Azure, Network, DdosProtectionPlan)

	//Storage item types
	StorageAccount       = shared.NewItemType(Azure, Storage, Account)
	StorageBlobContainer = shared.NewItemType(Azure, Storage, BlobContainer)
	StorageFileShare     = shared.NewItemType(Azure, Storage, FileShare)
	StorageTable         = shared.NewItemType(Azure, Storage, Table)
	StorageQueue         = shared.NewItemType(Azure, Storage, Queue)

	// SQL item types
	SQLDatabase                      = shared.NewItemType(Azure, SQL, Database)
	SQLRecoverableDatabase           = shared.NewItemType(Azure, SQL, RecoverableDatabase)
	SQLRecoveryServicesRecoveryPoint = shared.NewItemType(Azure, SQL, RecoveryServicesRecoveryPoint)
	SQLRestorableDroppedDatabase     = shared.NewItemType(Azure, SQL, RestorableDroppedDatabase)
	SQLServer                        = shared.NewItemType(Azure, SQL, Server)
	SQLElasticPool                   = shared.NewItemType(Azure, SQL, ElasticPool)

	// DBforPostgreSQL item types
	DBforPostgreSQLFlexibleServer = shared.NewItemType(Azure, DBforPostgreSQL, FlexibleServer)
	DBforPostgreSQLDatabase       = shared.NewItemType(Azure, DBforPostgreSQL, Database)

	// DocumentDB item types
	DocumentDBDatabaseAccounts          = shared.NewItemType(Azure, DocumentDB, DatabaseAccounts)
	DocumentDBPrivateEndpointConnection = shared.NewItemType(Azure, DocumentDB, PrivateEndpointConnection)

	// KeyVault item types
	KeyVaultVault      = shared.NewItemType(Azure, KeyVault, Vault)
	KeyVaultManagedHSM = shared.NewItemType(Azure, KeyVault, ManagedHSM)

	// ManagedIdentity item types
	ManagedIdentityUserAssignedIdentity = shared.NewItemType(Azure, ManagedIdentity, UserAssignedIdentity)

	// Batch item types
	BatchBatchAccount                   = shared.NewItemType(Azure, Batch, BatchAccount)
	BatchBatchApplication               = shared.NewItemType(Azure, Batch, BatchApplication)
	BatchBatchApplicationPackage        = shared.NewItemType(Azure, Batch, BatchApplicationPackage)
	BatchBatchPool                      = shared.NewItemType(Azure, Batch, BatchPool)
	BatchBatchCertificate               = shared.NewItemType(Azure, Batch, BatchCertificate)
	BatchBatchPrivateEndpointConnection = shared.NewItemType(Azure, Batch, BatchPrivateEndpointConnection)
	BatchBatchPrivateLinkResource       = shared.NewItemType(Azure, Batch, BatchPrivateLinkResource)
	BatchBatchDetector                  = shared.NewItemType(Azure, Batch, BatchDetector)
)
