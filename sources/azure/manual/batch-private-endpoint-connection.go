package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var BatchPrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.BatchBatchPrivateEndpointConnection)

type batchPrivateEndpointConnectionWrapper struct {
	client clients.BatchPrivateEndpointConnectionClient

	*azureshared.MultiResourceGroupBase
}

// NewBatchPrivateEndpointConnection returns a SearchableWrapper for Azure Batch private endpoint connections.
func NewBatchPrivateEndpointConnection(client clients.BatchPrivateEndpointConnectionClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &batchPrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.BatchBatchPrivateEndpointConnection,
		),
	}
}

func (b batchPrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: accountName and privateEndpointConnectionName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]
	connectionName := queryParts[1]

	if accountName == "" {
		return nil, azureshared.QueryError(errors.New("accountName cannot be empty"), scope, b.Type())
	}
	if connectionName == "" {
		return nil, azureshared.QueryError(errors.New("privateEndpointConnectionName cannot be empty"), scope, b.Type())
	}

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	resp, err := b.client.Get(ctx, rgScope.ResourceGroup, accountName, connectionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	item, sdpErr := b.azurePrivateEndpointConnectionToSDPItem(&resp.PrivateEndpointConnection, accountName, connectionName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}
	return item, nil
}

func (b batchPrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BatchAccountLookupByName,
		BatchPrivateEndpointConnectionLookupByName,
	}
}

func (b batchPrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: accountName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]

	if accountName == "" {
		return nil, azureshared.QueryError(errors.New("accountName cannot be empty"), scope, b.Type())
	}

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	pager := b.client.ListByBatchAccount(ctx, rgScope.ResourceGroup, accountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, b.Type())
		}

		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}

			item, sdpErr := b.azurePrivateEndpointConnectionToSDPItem(conn, accountName, *conn.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (b batchPrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: accountName"), scope, b.Type()))
		return
	}
	accountName := queryParts[0]

	if accountName == "" {
		stream.SendError(azureshared.QueryError(errors.New("accountName cannot be empty"), scope, b.Type()))
		return
	}

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, b.Type()))
		return
	}
	pager := b.client.ListByBatchAccount(ctx, rgScope.ResourceGroup, accountName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, b.Type()))
			return
		}
		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}
			item, sdpErr := b.azurePrivateEndpointConnectionToSDPItem(conn, accountName, *conn.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (b batchPrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BatchAccountLookupByName,
		},
	}
}

func (b batchPrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.BatchBatchAccount:      true,
		azureshared.NetworkPrivateEndpoint: true,
	}
}

func (b batchPrivateEndpointConnectionWrapper) azurePrivateEndpointConnectionToSDPItem(conn *armbatch.PrivateEndpointConnection, accountName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.BatchBatchPrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(conn.Tags),
	}

	// Health from provisioning state
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		switch *conn.Properties.ProvisioningState {
		case armbatch.PrivateEndpointConnectionProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armbatch.PrivateEndpointConnectionProvisioningStateCreating,
			armbatch.PrivateEndpointConnectionProvisioningStateUpdating,
			armbatch.PrivateEndpointConnectionProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armbatch.PrivateEndpointConnectionProvisioningStateFailed,
			armbatch.PrivateEndpointConnectionProvisioningStateCancelled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent Batch Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchAccount.String(),
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

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftbatch
func (b batchPrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Batch/batchAccounts/privateEndpointConnections/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/compute#azure-batch-account-reader
func (b batchPrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Azure Batch Account Reader"
}
