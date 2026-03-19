package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeDiskAccessPrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDiskAccessPrivateEndpointConnection)

type computeDiskAccessPrivateEndpointConnectionWrapper struct {
	client clients.ComputeDiskAccessPrivateEndpointConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewComputeDiskAccessPrivateEndpointConnection returns a SearchableWrapper for Azure disk access private endpoint connections.
func NewComputeDiskAccessPrivateEndpointConnection(client clients.ComputeDiskAccessPrivateEndpointConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeDiskAccessPrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ComputeDiskAccessPrivateEndpointConnection,
		),
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: diskAccessName and privateEndpointConnectionName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	diskAccessName := queryParts[0]
	connectionName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, diskAccessName, connectionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(&resp.PrivateEndpointConnection, diskAccessName, connectionName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}
	return item, nil
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskAccessLookupByName,
		ComputeDiskAccessPrivateEndpointConnectionLookupByName,
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: diskAccessName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	diskAccessName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.NewListPrivateEndpointConnectionsPager(rgScope.ResourceGroup, diskAccessName, nil)

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

			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, diskAccessName, *conn.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: diskAccessName"), scope, s.Type()))
		return
	}
	diskAccessName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.NewListPrivateEndpointConnectionsPager(rgScope.ResourceGroup, diskAccessName, nil)
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
			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, diskAccessName, *conn.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeDiskAccessLookupByName,
		},
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeDiskAccess:      true,
		azureshared.NetworkPrivateEndpoint: true,
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) azurePrivateEndpointConnectionToSDPItem(conn *armcompute.PrivateEndpointConnection, diskAccessName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(diskAccessName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeDiskAccessPrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health from provisioning state
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		switch *conn.Properties.ProvisioningState {
		case armcompute.PrivateEndpointConnectionProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armcompute.PrivateEndpointConnectionProvisioningStateCreating,
			armcompute.PrivateEndpointConnectionProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armcompute.PrivateEndpointConnectionProvisioningStateFailed:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent Disk Access
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeDiskAccess.String(),
			Method: sdp.QueryMethod_GET,
			Query:  diskAccessName,
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

func (s computeDiskAccessPrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/diskAccesses/privateEndpointConnections/read",
	}
}

func (s computeDiskAccessPrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
