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
	NetworkVirtualNetwork        = shared.NewItemType(Azure, Network, VirtualNetwork)
	NetworkSubnet                = shared.NewItemType(Azure, Network, Subnet)
	NetworkNetworkInterface      = shared.NewItemType(Azure, Network, NetworkInterface)
	NetworkPublicIPAddress       = shared.NewItemType(Azure, Network, PublicIPAddress)
	NetworkNetworkSecurityGroup  = shared.NewItemType(Azure, Network, NetworkSecurityGroup)
	NetworkVirtualNetworkPeering = shared.NewItemType(Azure, Network, VirtualNetworkPeering)
	//Storage item types
	StorageAccount       = shared.NewItemType(Azure, Storage, Account)
	StorageBlobContainer = shared.NewItemType(Azure, Storage, BlobContainer)
	StorageFileShare     = shared.NewItemType(Azure, Storage, FileShare)
	StorageTable         = shared.NewItemType(Azure, Storage, Table)
	StorageQueue         = shared.NewItemType(Azure, Storage, Queue)
)
