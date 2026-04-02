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

var DBforPostgreSQLFlexibleServerVirtualEndpointLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint)

type dbforPostgreSQLFlexibleServerVirtualEndpointWrapper struct {
	client clients.DBforPostgreSQLFlexibleServerVirtualEndpointClient

	*azureshared.MultiResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServerVirtualEndpoint(client clients.DBforPostgreSQLFlexibleServerVirtualEndpointClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &dbforPostgreSQLFlexibleServerVirtualEndpointWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/virtual-endpoints/get?view=rest-postgresql-2025-08-01
func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and virtualEndpointName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	virtualEndpointName := queryParts[1]
	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if virtualEndpointName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "virtualEndpointName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, virtualEndpointName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureVirtualEndpointToSDPItem(&resp.VirtualEndpoint, serverName, virtualEndpointName, scope)
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) azureVirtualEndpointToSDPItem(virtualEndpoint *armpostgresqlflexibleservers.VirtualEndpoint, serverName, virtualEndpointName, scope string) (*sdp.Item, *sdp.QueryError) {
	if virtualEndpoint.Name == nil {
		return nil, azureshared.QueryError(errors.New("virtual endpoint name is nil"), scope, s.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(virtualEndpoint)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, virtualEndpointName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            nil,
	}

	// Link to parent PostgreSQL Flexible Server
	if virtualEndpoint.ID != nil {
		params := azureshared.ExtractPathParamsFromResourceID(*virtualEndpoint.ID, []string{"flexibleServers"})
		if len(params) > 0 {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  params[0],
					Scope:  scope,
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

	// Link to member servers (Members field contains server names that this virtual endpoint can refer to)
	if virtualEndpoint.Properties != nil && virtualEndpoint.Properties.Members != nil {
		for _, memberServerName := range virtualEndpoint.Properties.Members {
			if memberServerName != nil && *memberServerName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *memberServerName,
						Scope:  scope,
					},
				})
			}
		}
	}

	// Link to virtual endpoint DNS names (VirtualEndpoints field contains DNS names)
	if virtualEndpoint.Properties != nil && virtualEndpoint.Properties.VirtualEndpoints != nil {
		for _, dnsName := range virtualEndpoint.Properties.VirtualEndpoints {
			if dnsName != nil && *dnsName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *dnsName,
						Scope:  "global",
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLFlexibleServerVirtualEndpointLookupByName,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/virtual-endpoints/list-by-server?view=rest-postgresql-2025-08-01
func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, virtualEndpoint := range page.Value {
			if virtualEndpoint.Name == nil {
				continue
			}
			item, sdpErr := s.azureVirtualEndpointToSDPItem(virtualEndpoint, serverName, *virtualEndpoint.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
		for _, virtualEndpoint := range page.Value {
			if virtualEndpoint.Name == nil {
				continue
			}
			item, sdpErr := s.azureVirtualEndpointToSDPItem(virtualEndpoint, serverName, *virtualEndpoint.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.DBforPostgreSQLFlexibleServer: true,
		stdlib.NetworkDNS:                         true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftdbforpostgresql
func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/virtualEndpoints/read",
	}
}

func (s dbforPostgreSQLFlexibleServerVirtualEndpointWrapper) PredefinedRole() string {
	return "Reader"
}
