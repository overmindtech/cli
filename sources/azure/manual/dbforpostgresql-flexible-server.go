package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var DBforPostgreSQLFlexibleServerLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServer)

type dbforPostgreSQLFlexibleServerWrapper struct {
	client clients.PostgreSQLFlexibleServersClient

	*azureshared.ResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServer(client clients.PostgreSQLFlexibleServersClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &dbforPostgreSQLFlexibleServerWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServer,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/servers/get?view=rest-postgresql-2025-08-01&tabs=HTTP
func (s dbforPostgreSQLFlexibleServerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: serverName"), scope, s.Type())
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName is empty"), scope, s.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}
	resp, err := s.client.Get(ctx, resourceGroup, serverName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureDBforPostgreSQLFlexibleServerToSDPItem(&resp.Server, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/servers/list-by-resource-group?view=rest-postgresql-2025-08-01&tabs=HTTP
func (s dbforPostgreSQLFlexibleServerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}
	pager := s.client.ListByResourceGroup(ctx, resourceGroup, nil)
	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}
		for _, server := range page.Value {
			if server.Name == nil {
				continue
			}
			item, sdpErr := s.azureDBforPostgreSQLFlexibleServerToSDPItem(server, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (s dbforPostgreSQLFlexibleServerWrapper) azureDBforPostgreSQLFlexibleServerToSDPItem(server *armpostgresqlflexibleservers.Server, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(server, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	if server.Name == nil {
		return nil, azureshared.QueryError(errors.New("serverName is nil"), scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            s.Type(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(server.Tags),
	}

	serverName := *server.Name

	// Link to Subnet (external resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}
	//
	// IMPORTANT: Subnets can be in a different resource group than the PostgreSQL Flexible Server.
	// We must extract the subscription ID and resource group from the subnet's resource ID to construct
	// the correct scope.
	if server.Properties != nil && server.Properties.Network != nil && server.Properties.Network.DelegatedSubnetResourceID != nil {
		subnetID := *server.Properties.Network.DelegatedSubnetResourceID
		// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}
		// Extract subscription, resource group, virtual network name, and subnet name
		scopeParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"subscriptions", "resourceGroups"})
		subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
		if len(scopeParams) >= 2 && len(subnetParams) >= 2 {
			subscriptionID := scopeParams[0]
			resourceGroupName := scopeParams[1]
			vnetName := subnetParams[0]
			subnetName := subnetParams[1]
			// Subnet adapter requires: resourceGroup, virtualNetworkName, subnetName
			// Use composite lookup key to join them
			query := shared.CompositeLookupKey(vnetName, subnetName)
			// Construct scope in format: {subscriptionID}.{resourceGroupName}
			// This ensures we query the correct resource group where the subnet actually exists
			scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkSubnet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  scope, // Use the subnet's scope, not the server's scope
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on subnet for network connectivity
					// If subnet is deleted/modified, server network access may be affected
					In:  true,
					Out: false,
				}, // Subnet is an external resource that the server depends on
			})

			// Link to Virtual Network (parent of subnet)
			// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get
			// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vnetName,
					Scope:  scope, // Use the same scope as the subnet
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on virtual network for network connectivity
					// If virtual network is deleted/modified, server network access may be affected
					In:  true,
					Out: false,
				}, // Virtual Network is an external resource that the server depends on
			})
		}
	}

	// Link to Databases (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/databases/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLDatabase.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Server changes (deletion, configuration, maintenance) directly affect database availability
			Out: false, // Database changes (schema, data) don't directly affect the server's configuration
		}, // Databases are child resources that depend on their parent server
	})

	// Link to Firewall Rules (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/firewall-rules/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/firewallRules?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerFirewallRule.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes affect firewall rules
			Out: true, // Firewall rule changes affect server connectivity
		}, // Firewall Rules are child resources that control server access
	})

	// Link to Configurations (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/configurations/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/configurations?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerConfiguration.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes affect configurations
			Out: true, // Configuration changes affect server behavior and performance
		}, // Configurations are child resources that control server settings
	})

	// Link to Fully Qualified Domain Name (DNS)
	// If the server has an FQDN, link it to the DNS standard library type
	if server.Properties != nil && server.Properties.FullyQualifiedDomainName != nil && *server.Properties.FullyQualifiedDomainName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *server.Properties.FullyQualifiedDomainName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // DNS changes affect server connectivity
				Out: true, // Server changes may affect DNS resolution
			}, // DNS names are shared resources that affect connectivity
		})
	}

	// Link to User Assigned Managed Identities (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if server.Identity != nil && server.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range server.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				// Extract scope from resource ID if it's in a different resource group
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// PostgreSQL Flexible Server depends on managed identity for authentication
						// If identity is deleted/modified, server operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to Private DNS Zone (external resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}
	if server.Properties != nil && server.Properties.Network != nil && server.Properties.Network.PrivateDNSZoneArmResourceID != nil {
		privateDNSZoneID := *server.Properties.Network.PrivateDNSZoneArmResourceID
		privateDNSZoneName := azureshared.ExtractResourceName(privateDNSZoneID)
		if privateDNSZoneName != "" {
			// Extract scope from resource ID if it's in a different resource group
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(privateDNSZoneID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkPrivateDNSZone.String(),
					Method: sdp.QueryMethod_GET,
					Query:  privateDNSZoneName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on private DNS zone for DNS resolution
					// If DNS zone is deleted/modified, server DNS resolution may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Administrators (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/administrators/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/administrators?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerAdministrator.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes affect administrators
			Out: true, // Administrator changes affect server access and authentication
		}, // Administrators are child resources that control server access
	})

	// Link to Private Endpoint Connections (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/private-endpoint-connections/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/privateEndpointConnections?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes affect private endpoint connections
			Out: true, // Private endpoint connection changes affect server network connectivity
		}, // Private Endpoint Connections are child resources that manage private network access
	})

	// Link to Network Private Endpoints (external resources) from PrivateEndpointConnections
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateEndpoints/{privateEndpointName}
	if server.Properties != nil && server.Properties.PrivateEndpointConnections != nil {
		for _, peConnection := range server.Properties.PrivateEndpointConnections {
			if peConnection.Properties != nil && peConnection.Properties.PrivateEndpoint != nil && peConnection.Properties.PrivateEndpoint.ID != nil {
				privateEndpointID := *peConnection.Properties.PrivateEndpoint.ID
				privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
				if privateEndpointName != "" {
					// Extract scope from resource ID if it's in a different resource group
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  privateEndpointName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Private endpoint changes (deletion, network configuration) affect the PostgreSQL Flexible Server's private connectivity
							// Server deletion or configuration changes may affect the private endpoint's connection state
							In:  true,
							Out: true,
						}, // Private endpoints are tightly coupled to the server - changes affect connectivity
					})
				}
			}
		}
	}

	// Link to Private Link Resources (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/private-link-resources/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/privateLinkResources?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerPrivateLinkResource.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes affect private link resources
			Out: true, // Private link resource changes affect server private connectivity
		}, // Private Link Resources are child resources that define available private link services
	})

	// Link to Replicas (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/replicas/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/replicas?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerReplica.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Server changes (deletion, configuration) directly affect replica availability
			Out: false, // Replica changes don't directly affect the primary server's configuration
		}, // Replicas are child resources that depend on their parent server
	})

	// Link to Migrations (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/migrations/list-by-target-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/migrations?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerMigration.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Server changes affect migration operations
			Out: false, // Migration changes don't directly affect the server's configuration
		}, // Migrations are child resources that represent data migration operations
	})

	// Link to Backups (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/backups/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/backups?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerBackup.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Server changes (deletion, configuration) directly affect backup availability
			Out: false, // Backup changes don't directly affect the server's configuration
		}, // Backups are child resources that depend on their parent server
	})

	// Link to Virtual Endpoints (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/virtual-endpoints/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/virtualEndpoints?api-version=2025-08-01
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Server changes (deletion, configuration) directly affect virtual endpoint availability
			Out: true, // Virtual endpoint changes affect server connectivity and routing
		}, // Virtual Endpoints are child resources that control server network access
	})

	// Link to Key Vault Vault (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.PrimaryKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.PrimaryKeyURI
		// Key URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
		// Extract vault name from URI
		if vaultName := azureshared.ExtractVaultNameFromURI(keyURI); vaultName != "" {
			// Key Vault can be in a different resource group, but we don't have that info from the URI
			// Use default scope and let the Key Vault adapter handle cross-resource-group lookups if needed
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on Key Vault for customer-managed encryption keys
					// If Key Vault is deleted/modified, server encryption may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Key Vault Key (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keys/get-key
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}/keys/{keyName}
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.PrimaryKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.PrimaryKeyURI
		// Key URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
		// Extract vault name and key name from URI
		vaultName := azureshared.ExtractVaultNameFromURI(keyURI)
		keyName := azureshared.ExtractKeyNameFromURI(keyURI)
		if vaultName != "" && keyName != "" {
			// Use composite lookup key for vault name and key name
			query := shared.CompositeLookupKey(vaultName, keyName)
			// Key Vault can be in a different resource group, but we don't have that info from the URI
			// Use default scope and let the Key Vault adapter handle cross-resource-group lookups if needed
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on Key Vault key for customer-managed encryption
					// If key is deleted/modified, server encryption operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Primary User Assigned Managed Identity (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.PrimaryUserAssignedIdentityID != nil {
		identityID := *server.Properties.DataEncryption.PrimaryUserAssignedIdentityID
		identityName := azureshared.ExtractResourceName(identityID)
		if identityName != "" {
			// Extract scope from resource ID if it's in a different resource group
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(identityID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  identityName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on managed identity for accessing encryption keys
					// If identity is deleted/modified, server encryption operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Geo Backup Key Vault Vault (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.GeoBackupKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.GeoBackupKeyURI
		// Key URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
		// Extract vault name from URI
		if vaultName := azureshared.ExtractVaultNameFromURI(keyURI); vaultName != "" {
			// Key Vault can be in a different resource group, but we don't have that info from the URI
			// Use default scope and let the Key Vault adapter handle cross-resource-group lookups if needed
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on Key Vault for geo-redundant backup encryption keys
					// If Key Vault is deleted/modified, server geo-backup encryption may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Geo Backup Key Vault Key (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keys/get-key
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}/keys/{keyName}
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.GeoBackupKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.GeoBackupKeyURI
		// Key URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
		// Extract vault name and key name from URI
		vaultName := azureshared.ExtractVaultNameFromURI(keyURI)
		keyName := azureshared.ExtractKeyNameFromURI(keyURI)
		if vaultName != "" && keyName != "" {
			// Use composite lookup key for vault name and key name
			query := shared.CompositeLookupKey(vaultName, keyName)
			// Key Vault can be in a different resource group, but we don't have that info from the URI
			// Use default scope and let the Key Vault adapter handle cross-resource-group lookups if needed
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on Key Vault key for geo-redundant backup encryption
					// If key is deleted/modified, server geo-backup encryption operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Geo Backup User Assigned Managed Identity (external resource) from Data Encryption
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.GeoBackupUserAssignedIdentityID != nil {
		identityID := *server.Properties.DataEncryption.GeoBackupUserAssignedIdentityID
		identityName := azureshared.ExtractResourceName(identityID)
		if identityName != "" {
			// Extract scope from resource ID if it's in a different resource group
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(identityID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  identityName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// PostgreSQL Flexible Server depends on managed identity for accessing geo-backup encryption keys
					// If identity is deleted/modified, server geo-backup encryption operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Source Server (for replica servers and point-in-time restore servers)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/servers/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}
	if server.Properties != nil && server.Properties.SourceServerResourceID != nil {
		sourceServerID := *server.Properties.SourceServerResourceID
		sourceServerName := azureshared.ExtractResourceName(sourceServerID)
		if sourceServerName != "" {
			// Extract scope from resource ID if it's in a different resource group
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(sourceServerID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  sourceServerName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Replica server depends on source server for replication
					// If source server is deleted/modified, replica operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	return sdpItem, nil
}

func (s dbforPostgreSQLFlexibleServerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
	}
}

func (s dbforPostgreSQLFlexibleServerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkPrivateDNSZone,
		azureshared.NetworkPrivateEndpoint,
		azureshared.DBforPostgreSQLDatabase,
		azureshared.DBforPostgreSQLFlexibleServerFirewallRule,
		azureshared.DBforPostgreSQLFlexibleServerConfiguration,
		azureshared.DBforPostgreSQLFlexibleServerAdministrator,
		azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection,
		azureshared.DBforPostgreSQLFlexibleServerPrivateLinkResource,
		azureshared.DBforPostgreSQLFlexibleServerReplica,
		azureshared.DBforPostgreSQLFlexibleServerMigration,
		azureshared.DBforPostgreSQLFlexibleServerBackup,
		azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint,
		azureshared.DBforPostgreSQLFlexibleServer, // For replica-to-source server relationship
		stdlib.NetworkDNS,
		azureshared.ManagedIdentityUserAssignedIdentity,
		azureshared.KeyVaultVault,
		azureshared.KeyVaultKey,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/postgresql_flexible_server
func (s dbforPostgreSQLFlexibleServerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_postgresql_flexible_server.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/databases#microsoftdbforpostgresql
func (s dbforPostgreSQLFlexibleServerWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/read",
	}
}

func (s dbforPostgreSQLFlexibleServerWrapper) PredefinedRole() string {
	return "Reader"
}
