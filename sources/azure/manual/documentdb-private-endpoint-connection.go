package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var DocumentDBPrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.DocumentDBPrivateEndpointConnection)

type documentDBPrivateEndpointConnectionWrapper struct {
	client clients.DocumentDBPrivateEndpointConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewDocumentDBPrivateEndpointConnection returns a SearchableWrapper for Azure Cosmos DB (DocumentDB) database account private endpoint connections.
func NewDocumentDBPrivateEndpointConnection(client clients.DocumentDBPrivateEndpointConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &documentDBPrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DocumentDBPrivateEndpointConnection,
		),
	}
}

func (s documentDBPrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: accountName and privateEndpointConnectionName",
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

func (s documentDBPrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DocumentDBDatabaseAccountsLookupByName,
		DocumentDBPrivateEndpointConnectionLookupByName,
	}
}

func (s documentDBPrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: accountName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByDatabaseAccount(ctx, rgScope.ResourceGroup, accountName)

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

func (s documentDBPrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: accountName"), scope, s.Type()))
		return
	}
	accountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByDatabaseAccount(ctx, rgScope.ResourceGroup, accountName)
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

func (s documentDBPrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DocumentDBDatabaseAccountsLookupByName,
		},
	}
}

func (s documentDBPrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.DocumentDBDatabaseAccounts: true,
		azureshared.NetworkPrivateEndpoint:     true,
	}
}

func (s documentDBPrivateEndpointConnectionWrapper) azurePrivateEndpointConnectionToSDPItem(conn *armcosmos.PrivateEndpointConnection, accountName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DocumentDBPrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health from provisioning state (Cosmos uses *string, not an enum)
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		state := strings.ToLower(*conn.Properties.ProvisioningState)
		switch state {
		case "succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "creating", "deleting":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "failed":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent DocumentDB Database Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DocumentDBDatabaseAccounts.String(),
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

func (s documentDBPrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DocumentDB/databaseAccounts/privateEndpointConnections/read",
	}
}

func (s documentDBPrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
