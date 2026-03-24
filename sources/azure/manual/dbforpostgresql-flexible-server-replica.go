package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var DBforPostgreSQLFlexibleServerReplicaLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServerReplica)

type dbforPostgreSQLFlexibleServerReplicaWrapper struct {
	client clients.DBforPostgreSQLFlexibleServerReplicaClient

	*azureshared.MultiResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServerReplica(client clients.DBforPostgreSQLFlexibleServerReplicaClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &dbforPostgreSQLFlexibleServerReplicaWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServerReplica,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/servers/get
func (s dbforPostgreSQLFlexibleServerReplicaWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and replicaName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	replicaName := queryParts[1]
	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if replicaName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "replicaName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, replicaName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureReplicaToSDPItem(&resp.Server, serverName, replicaName, scope)
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) azureReplicaToSDPItem(server *armpostgresqlflexibleservers.Server, serverName, replicaName, scope string) (*sdp.Item, *sdp.QueryError) {
	if server.Name == nil {
		return nil, azureshared.QueryError(errors.New("replica name is nil"), scope, s.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(server, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, replicaName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DBforPostgreSQLFlexibleServerReplica.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(server.Tags),
	}

	// Map provisioning state to health
	if server.Properties != nil && server.Properties.State != nil {
		switch *server.Properties.State {
		case armpostgresqlflexibleservers.ServerStateReady:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armpostgresqlflexibleservers.ServerStateStarting, armpostgresqlflexibleservers.ServerStateStopping, armpostgresqlflexibleservers.ServerStateUpdating, armpostgresqlflexibleservers.ServerStateProvisioning, armpostgresqlflexibleservers.ServerStateRestarting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armpostgresqlflexibleservers.ServerStateDisabled, armpostgresqlflexibleservers.ServerStateStopped, armpostgresqlflexibleservers.ServerStateInaccessible:
			sdpItem.Health = sdp.Health_HEALTH_WARNING.Enum()
		case armpostgresqlflexibleservers.ServerStateDropping:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	} else {
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	// Link to parent PostgreSQL Flexible Server (source server for replica)
	if server.Properties != nil && server.Properties.SourceServerResourceID != nil {
		sourceServerID := *server.Properties.SourceServerResourceID
		sourceServerName := azureshared.ExtractResourceName(sourceServerID)
		if sourceServerName != "" {
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
			})
		}
	} else {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
				Method: sdp.QueryMethod_GET,
				Query:  serverName,
				Scope:  scope,
			},
		})
	}

	// Link to Fully Qualified Domain Name (DNS)
	if server.Properties != nil && server.Properties.FullyQualifiedDomainName != nil && *server.Properties.FullyQualifiedDomainName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *server.Properties.FullyQualifiedDomainName,
				Scope:  "global",
			},
		})
	}

	// Link to Subnet (external resource)
	if server.Properties != nil && server.Properties.Network != nil && server.Properties.Network.DelegatedSubnetResourceID != nil {
		subnetID := *server.Properties.Network.DelegatedSubnetResourceID
		scopeParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"subscriptions", "resourceGroups"})
		subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
		if len(scopeParams) >= 2 && len(subnetParams) >= 2 {
			subscriptionID := scopeParams[0]
			resourceGroupName := scopeParams[1]
			vnetName := subnetParams[0]
			subnetName := subnetParams[1]
			query := shared.CompositeLookupKey(vnetName, subnetName)
			linkedScope := subscriptionID + "." + resourceGroupName
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkSubnet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  linkedScope,
				},
			})

			// Link to Virtual Network (parent of subnet)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vnetName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Link to Private DNS Zone (external resource)
	if server.Properties != nil && server.Properties.Network != nil && server.Properties.Network.PrivateDNSZoneArmResourceID != nil {
		privateDNSZoneID := *server.Properties.Network.PrivateDNSZoneArmResourceID
		privateDNSZoneName := azureshared.ExtractResourceName(privateDNSZoneID)
		if privateDNSZoneName != "" {
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
			})
		}
	}

	// Link to User Assigned Managed Identities
	if server.Identity != nil && server.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range server.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
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
				})
			}
		}
	}

	// Link to Network Private Endpoints from PrivateEndpointConnections
	if server.Properties != nil && server.Properties.PrivateEndpointConnections != nil {
		for _, peConnection := range server.Properties.PrivateEndpointConnections {
			if peConnection.Properties != nil && peConnection.Properties.PrivateEndpoint != nil && peConnection.Properties.PrivateEndpoint.ID != nil {
				privateEndpointID := *peConnection.Properties.PrivateEndpoint.ID
				privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
				if privateEndpointName != "" {
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
					})
				}
			}
		}
	}

	// Link to Key Vault Vault from Data Encryption (Primary Key)
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.PrimaryKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.PrimaryKeyURI
		if vaultName := azureshared.ExtractVaultNameFromURI(keyURI); vaultName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  scope,
				},
			})

			// Link to Key Vault Key
			keyName := azureshared.ExtractKeyNameFromURI(keyURI)
			if keyName != "" {
				query := shared.CompositeLookupKey(vaultName, keyName)
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  query,
						Scope:  scope,
					},
				})
			}
		}
	}

	// Link to Primary User Assigned Managed Identity from Data Encryption
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.PrimaryUserAssignedIdentityID != nil {
		identityID := *server.Properties.DataEncryption.PrimaryUserAssignedIdentityID
		identityName := azureshared.ExtractResourceName(identityID)
		if identityName != "" {
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
			})
		}
	}

	// Link to Geo Backup Key Vault Vault from Data Encryption
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.GeoBackupKeyURI != nil {
		keyURI := *server.Properties.DataEncryption.GeoBackupKeyURI
		if vaultName := azureshared.ExtractVaultNameFromURI(keyURI); vaultName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  scope,
				},
			})

			// Link to Geo Backup Key Vault Key
			keyName := azureshared.ExtractKeyNameFromURI(keyURI)
			if keyName != "" {
				query := shared.CompositeLookupKey(vaultName, keyName)
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  query,
						Scope:  scope,
					},
				})
			}
		}
	}

	// Link to Geo Backup User Assigned Managed Identity from Data Encryption
	if server.Properties != nil && server.Properties.DataEncryption != nil && server.Properties.DataEncryption.GeoBackupUserAssignedIdentityID != nil {
		identityID := *server.Properties.DataEncryption.GeoBackupUserAssignedIdentityID
		identityName := azureshared.ExtractResourceName(identityID)
		if identityName != "" {
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
			})
		}
	}

	return sdpItem, nil
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLFlexibleServerReplicaLookupByName,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/replicas/list-by-server
func (s dbforPostgreSQLFlexibleServerReplicaWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)

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
			item, sdpErr := s.azureReplicaToSDPItem(server, serverName, *server.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, s.Type()))
		return
	}
	serverName := queryParts[0]
	if serverName == "" {
		stream.SendError(azureshared.QueryError(errors.New("serverName cannot be empty"), scope, s.Type()))
		return
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, server := range page.Value {
			if server.Name == nil {
				continue
			}
			item, sdpErr := s.azureReplicaToSDPItem(server, serverName, *server.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.DBforPostgreSQLFlexibleServer,
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkPrivateDNSZone,
		azureshared.NetworkPrivateEndpoint,
		azureshared.ManagedIdentityUserAssignedIdentity,
		azureshared.KeyVaultVault,
		azureshared.KeyVaultKey,
		stdlib.NetworkDNS,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftdbforpostgresql
func (s dbforPostgreSQLFlexibleServerReplicaWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/read",
		"Microsoft.DBforPostgreSQL/flexibleServers/replicas/read",
	}
}

func (s dbforPostgreSQLFlexibleServerReplicaWrapper) PredefinedRole() string {
	return "Reader"
}
