package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkRouteLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkRoute)

type networkRouteWrapper struct {
	client clients.RoutesClient
	*azureshared.MultiResourceGroupBase
}

// NewNetworkRoute creates a new networkRouteWrapper instance (SearchableWrapper: child of route table).
func NewNetworkRoute(client clients.RoutesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkRouteWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkRoute,
		),
	}
}

func (n networkRouteWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: routeTableName and routeName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	routeTableName := queryParts[0]
	routeName := queryParts[1]
	if routeName == "" {
		return nil, azureshared.QueryError(errors.New("route name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, routeTableName, routeName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureRouteToSDPItem(&resp.Route, routeTableName, routeName, scope)
}

func (n networkRouteWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkRouteTableLookupByName,
		NetworkRouteLookupByUniqueAttr,
	}
}

func (n networkRouteWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: routeTableName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	routeTableName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, routeTableName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, route := range page.Value {
			if route == nil || route.Name == nil {
				continue
			}
			item, sdpErr := n.azureRouteToSDPItem(route, routeTableName, *route.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkRouteWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: routeTableName"), scope, n.Type()))
		return
	}
	routeTableName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, routeTableName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, route := range page.Value {
			if route == nil || route.Name == nil {
				continue
			}
			item, sdpErr := n.azureRouteToSDPItem(route, routeTableName, *route.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkRouteWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{NetworkRouteTableLookupByName},
	}
}

func (n networkRouteWrapper) azureRouteToSDPItem(route *armnetwork.Route, routeTableName, routeName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(route, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(routeTableName, routeName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkRoute.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health status from ProvisioningState
	if route.Properties != nil && route.Properties.ProvisioningState != nil {
		switch *route.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateCreating, armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	// Link to parent Route Table
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkRouteTable.String(),
			Method: sdp.QueryMethod_GET,
			Query:  routeTableName,
			Scope:  scope,
		},
	})

	// Link to NextHopIPAddress (IP address to stdlib)
	if route.Properties != nil && route.Properties.NextHopIPAddress != nil && *route.Properties.NextHopIPAddress != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  *route.Properties.NextHopIPAddress,
				Scope:  "global",
			},
		})
	}

	return sdpItem, nil
}

func (n networkRouteWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkRouteTable,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route
func (n networkRouteWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{TerraformMethod: sdp.QueryMethod_SEARCH, TerraformQueryMap: "azurerm_route.id"},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions-reference#microsoftnetwork
func (n networkRouteWrapper) IAMPermissions() []string {
	return []string{"Microsoft.Network/routeTables/routes/read"}
}

func (n networkRouteWrapper) PredefinedRole() string {
	return "Reader"
}
