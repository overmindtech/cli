package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var SQLServerPrivateEndpointConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServerPrivateEndpointConnection)

type sqlServerPrivateEndpointConnectionWrapper struct {
	client clients.SQLServerPrivateEndpointConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewSQLServerPrivateEndpointConnection returns a SearchableWrapper for Azure SQL server private endpoint connections.
func NewSQLServerPrivateEndpointConnection(client clients.SQLServerPrivateEndpointConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlServerPrivateEndpointConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServerPrivateEndpointConnection,
		),
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and privateEndpointConnectionName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	connectionName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, connectionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(&resp.PrivateEndpointConnection, serverName, connectionName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}
	return item, nil
}

func (s sqlServerPrivateEndpointConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLServerPrivateEndpointConnectionLookupByName,
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]

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

		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}

			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, serverName, *conn.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlServerPrivateEndpointConnectionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, s.Type()))
		return
	}
	serverName := queryParts[0]

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
		for _, conn := range page.Value {
			if conn == nil || conn.Name == nil {
				continue
			}
			item, sdpErr := s.azurePrivateEndpointConnectionToSDPItem(conn, serverName, *conn.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLServer:              true,
		azureshared.NetworkPrivateEndpoint: true,
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) azurePrivateEndpointConnectionToSDPItem(conn *armsql.PrivateEndpointConnection, serverName, connectionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(conn)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, connectionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServerPrivateEndpointConnection.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health from provisioning state (armsql uses PrivateEndpointProvisioningState enum)
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		state := strings.ToLower(string(*conn.Properties.ProvisioningState))
		switch state {
		case "ready":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "approving", "dropping":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "failed", "rejecting":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent SQL Server
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.SQLServer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  serverName,
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

func (s sqlServerPrivateEndpointConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/privateEndpointConnections/read",
	}
}

func (s sqlServerPrivateEndpointConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
