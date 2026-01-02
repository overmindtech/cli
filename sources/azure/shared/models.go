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

	// Storage
	Storage shared.API = "storage" // Microsoft.Storage

	// SQL
	SQL shared.API = "sql" // Microsoft.Sql

	// DocumentDB
	DocumentDB shared.API = "documentdb" // Microsoft.DocumentDB

	// KeyVault
	KeyVault shared.API = "keyvault" // Microsoft.KeyVault

	// ManagedIdentity
	ManagedIdentity shared.API = "managedidentity" // Microsoft.ManagedIdentity

	// DBforPostgreSQL
	DBforPostgreSQL shared.API = "dbforpostgresql" // Microsoft.DBforPostgreSQL
)

// Resources
// These represent the actual resource types within each Azure resource provider
const (
	// Compute resources
	VirtualMachine           shared.Resource = "virtual-machine"
	Disk                     shared.Resource = "disk"
	AvailabilitySet          shared.Resource = "availability-set"
	VirtualMachineExtension  shared.Resource = "virtual-machine-extension"
	VirtualMachineRunCommand shared.Resource = "virtual-machine-run-command"

	// Network resources
	VirtualNetwork                      shared.Resource = "virtual-network"
	Subnet                              shared.Resource = "subnet"
	NetworkInterface                    shared.Resource = "network-interface"
	PublicIPAddress                     shared.Resource = "public-ip-address"
	NetworkSecurityGroup                shared.Resource = "network-security-group"
	VirtualNetworkPeering               shared.Resource = "virtual-network-peering"
	NetworkInterfaceIPConfiguration     shared.Resource = "network-interface-ip-configuration"
	PrivateEndpoint                     shared.Resource = "private-endpoint"
	LoadBalancer                        shared.Resource = "load-balancer"
	LoadBalancerFrontendIPConfiguration shared.Resource = "load-balancer-frontend-ip-configuration"
	LoadBalancerBackendAddressPool      shared.Resource = "load-balancer-backend-address-pool"
	LoadBalancerInboundNatRule          shared.Resource = "load-balancer-inbound-nat-rule"
	LoadBalancerLoadBalancingRule       shared.Resource = "load-balancer-load-balancing-rule"
	LoadBalancerProbe                   shared.Resource = "load-balancer-probe"
	LoadBalancerOutboundRule            shared.Resource = "load-balancer-outbound-rule"
	LoadBalancerInboundNatPool          shared.Resource = "load-balancer-inbound-nat-pool"
	PublicIPPrefix                  shared.Resource = "public-ip-prefix"
	NatGateway                      shared.Resource = "nat-gateway"
	DdosProtectionPlan              shared.Resource = "ddos-protection-plan"

	// Storage resources
	Account       shared.Resource = "account"
	BlobContainer shared.Resource = "blob-container"
	FileShare     shared.Resource = "file-share"
	Table         shared.Resource = "table"
	Queue         shared.Resource = "queue"

	// SQL resources
	Database                      shared.Resource = "database"
	RecoverableDatabase           shared.Resource = "recoverable-database"
	RestorableDroppedDatabase     shared.Resource = "restorable-dropped-database"
	RecoveryServicesRecoveryPoint shared.Resource = "recovery-services-recovery-point"
	Server                        shared.Resource = "server"
	ElasticPool                   shared.Resource = "elastic-pool"

	// DBforPostgreSQL resources
	FlexibleServer shared.Resource = "flexible-server"

	// DocumentDB resources
	DatabaseAccounts          shared.Resource = "database-accounts"
	PrivateEndpointConnection shared.Resource = "private-endpoint-connection"

	// KeyVault resources
	Vault      shared.Resource = "vault"
	ManagedHSM shared.Resource = "managed-hsm"

	// ManagedIdentity resources
	UserAssignedIdentity shared.Resource = "user-assigned-identity"
)
