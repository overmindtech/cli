package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

const Azure shared.Source = "azure"

// APIs (Azure Resource Provider namespaces)
// Azure organizes resources by resource providers (e.g., Microsoft.Compute, Microsoft.Network)
// We use simplified names following the same pattern as GCP
// Reference: https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/azure-services-resource-providers
const (
	// Compute
	Compute shared.API = "compute" // Microsoft.Compute

	// Networking
	Network shared.API = "network" // Microsoft.Network
)

// Resources
// These represent the actual resource types within each Azure resource provider
const (
	// Compute resources
	VirtualMachine  shared.Resource = "virtual-machine"
	Disk            shared.Resource = "disk"
	AvailabilitySet shared.Resource = "availability-set"

	// Network resources
	VirtualNetwork       shared.Resource = "virtual-network"
	Subnet               shared.Resource = "subnet"
	NetworkInterface     shared.Resource = "network-interface"
	PublicIPAddress      shared.Resource = "public-ip-address"
	NetworkSecurityGroup shared.Resource = "network-security-group"
)
