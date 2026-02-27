package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var StoragePrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.StoragePrivateEndpointConnection)

type storagePrivateEndpointConnectionWrapper struct {
	client clients.StoragePrivateEndpointConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewStoragePrivateEndpointConnection returns a SearchableWrapper for Azure storage account private endpoint connections.
func NewStoragePrivateEndpointConnection(client clients.StoragePrivateEndpointConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &storagePrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StoragePrivateEndpointConnection,
		),
	}
}

func (s storagePrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and privateEndpointConnectionName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]
	connectionName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, accountName, connectionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(&resp.PrivateEndpointConnection, accountName, connectionName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}
	return item, nil
}

func (s storagePrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StoragePrivateEndpointConnectionLookupByName,
	}
}

func (s storagePrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: storageAccountName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.List(ctx, rgScope.ResourceGroup, accountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, conn := range page.Value {
			if conn.Name == nil {
				continue
			}

			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, accountName, *conn.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storagePrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: storageAccountName"), scope, s.Type()))
		return
	}
	accountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.List(ctx, rgScope.ResourceGroup, accountName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, conn := range page.Value {
			if conn.Name == nil {
				continue
			}
			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, accountName, *conn.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s storagePrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName,
		},
	}
}

func (s storagePrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.StorageAccount:         true,
		azureshared.NetworkPrivateEndpoint: true,
	}
}

func (s storagePrivateEndpointConnectionWrapper) azurePrivateEndpointConnectionToSDPItem(conn *armstorage.PrivateEndpointConnection, accountName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StoragePrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health from provisioning state
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		switch *conn.Properties.ProvisioningState {
		case armstorage.PrivateEndpointConnectionProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armstorage.PrivateEndpointConnectionProvisioningStateCreating,
			armstorage.PrivateEndpointConnectionProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armstorage.PrivateEndpointConnectionProvisioningStateFailed:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent Storage Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  accountName,
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

	return sdpItem, nil
}

func (s storagePrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/privateEndpointConnections/read",
	}
}

func (s storagePrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
