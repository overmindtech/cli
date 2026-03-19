package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var KeyVaultManagedHSMPrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.KeyVaultManagedHSMPrivateEndpointConnection)

type keyvaultManagedHSMPrivateEndpointConnectionWrapper struct {
	client clients.KeyVaultManagedHSMPrivateEndpointConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewKeyVaultManagedHSMPrivateEndpointConnection returns a SearchableWrapper for Azure Key Vault Managed HSM private endpoint connections.
func NewKeyVaultManagedHSMPrivateEndpointConnection(client clients.KeyVaultManagedHSMPrivateEndpointConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &keyvaultManagedHSMPrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.KeyVaultManagedHSMPrivateEndpointConnection,
		),
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: hsmName and privateEndpointConnectionName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	hsmName := queryParts[0]
	connectionName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, hsmName, connectionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item, sdpErr := s.azureMHSMPrivateEndpointConnectionToSDPItem(&resp.MHSMPrivateEndpointConnection, hsmName, connectionName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}
	return item, nil
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		KeyVaultManagedHSMsLookupByName,
		KeyVaultManagedHSMPrivateEndpointConnectionLookupByName,
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: hsmName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	hsmName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByResource(ctx, rgScope.ResourceGroup, hsmName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}

			item, sdpErr := s.azureMHSMPrivateEndpointConnectionToSDPItem(conn, hsmName, *conn.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: hsmName"), scope, s.Type()))
		return
	}
	hsmName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByResource(ctx, rgScope.ResourceGroup, hsmName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}
			item, sdpErr := s.azureMHSMPrivateEndpointConnectionToSDPItem(conn, hsmName, *conn.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			KeyVaultManagedHSMsLookupByName,
		},
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.KeyVaultManagedHSM:                  true,
		azureshared.NetworkPrivateEndpoint:              true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) azureMHSMPrivateEndpointConnectionToSDPItem(conn *armkeyvault.MHSMPrivateEndpointConnection, hsmName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(hsmName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.KeyVaultManagedHSMPrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(conn.Tags),
	}

	// Health from provisioning state
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		state := strings.ToLower(string(*conn.Properties.ProvisioningState))
		switch state {
		case "succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "creating", "updating", "deleting":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "failed":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent Key Vault Managed HSM
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.KeyVaultManagedHSM.String(),
			Method: sdp.QueryMethod_GET,
			Query:  hsmName,
			Scope:  scope,
		},
	})

	// Link to Network Private Endpoint when present (may be in different resource group)
	if conn.Properties != nil && conn.Properties.PrivateEndpoint != nil && conn.Properties.PrivateEndpoint.ID != nil {
		peID := *conn.Properties.PrivateEndpoint.ID
		peName := azureshared.ExtractResourceName(peID)
		if peName != "" {
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(peID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkPrivateEndpoint.String(),
					Method: sdp.QueryMethod_GET,
					Query:  peName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Link to User Assigned Managed Identities (same pattern as KeyVaultManagedHSM adapter)
	// User Assigned Identities can be in a different resource group than the Managed HSM.
	if conn.Identity != nil && conn.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range conn.Identity.UserAssignedIdentities {
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

	return sdpItem, nil
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.KeyVault/managedHSMs/privateEndpointConnections/read",
	}
}

func (s keyvaultManagedHSMPrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
