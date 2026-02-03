package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkRouteTableLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkRouteTable)

type networkRouteTableWrapper struct {
	client clients.RouteTablesClient

	*azureshared.ResourceGroupBase
}

func NewNetworkRouteTable(client clients.RouteTablesClient, subscriptionID, resourceGroup string) *networkRouteTableWrapper {
	return &networkRouteTableWrapper{
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkRouteTable,
		),
		client: client,
	}
}

func (n networkRouteTableWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	pager := n.client.List(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}
		for _, routeTable := range page.Value {
			if routeTable.Name == nil {
				continue
			}
			item, sdpErr := n.azureRouteTableToSDPItem(routeTable)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkRouteTableWrapper) azureRouteTableToSDPItem(routeTable *armnetwork.RouteTable) (*sdp.Item, *sdp.QueryError) {
	if routeTable.Name == nil {
		return nil, azureshared.QueryError(errors.New("route table name is nil"), n.DefaultScope(), n.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(routeTable, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	routeTableName := *routeTable.Name

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkRouteTable.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(routeTable.Tags),
	}

	// Link to Routes (child resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/routes/get
	if routeTable.Properties != nil && routeTable.Properties.Routes != nil {
		for _, route := range routeTable.Properties.Routes {
			if route != nil && route.Name != nil && *route.Name != "" {
				// Routes are child resources accessed via: routeTables/{routeTableName}/routes/{routeName}
				// Query requires routeTableName and routeName
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkRoute.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(routeTableName, *route.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Route changes affect the route table's routing behavior
						Out: false, // Route table changes don't affect individual routes (routes are part of the table)
					},
				})

				// Link to NextHopIPAddress (IP address to stdlib)
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/routes/get
				if route.Properties != nil && route.Properties.NextHopIPAddress != nil && *route.Properties.NextHopIPAddress != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  *route.Properties.NextHopIPAddress,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// IPs are always linked bidirectionally
							In:  true,
							Out: true,
						},
					})
				}

				// Note: We don't link to VirtualNetworkGateway when nextHopType is VirtualNetworkGateway
				// because the Route struct doesn't contain a direct gateway ID. The gateway name would need
				// to be derivable from the route or searched, but typically VirtualNetworkGateway routes
				// don't have nextHopIPAddress. This link will be implemented when we can determine how to
				// identify the gateway from the route.
				// Reference: https://learn.microsoft.com/en-us/rest/api/network/virtual-network-gateways/get
			}
		}
	}

	// Link to Subnets (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	if routeTable.Properties != nil && routeTable.Properties.Subnets != nil {
		for _, subnetRef := range routeTable.Properties.Subnets {
			if subnetRef != nil && subnetRef.ID != nil {
				subnetID := *subnetRef.ID
				// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
				// Extract virtual network name and subnet name using helper function
				subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
				if len(subnetParams) >= 2 {
					vnetName := subnetParams[0]
					subnetName := subnetParams[1]
					scope := n.DefaultScope()
					// Check if subnet is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vnetName, subnetName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Subnet changes (like route table association) affect the route table's usage
							Out: true, // Route table changes affect traffic routing in the subnet
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkRouteTableWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the route table name"), n.DefaultScope(), n.Type())
	}
	routeTableName := queryParts[0]
	if routeTableName == "" {
		return nil, azureshared.QueryError(errors.New("route table name is empty"), n.DefaultScope(), n.Type())
	}
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	resp, err := n.client.Get(ctx, resourceGroup, routeTableName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}
	return n.azureRouteTableToSDPItem(&resp.RouteTable)
}

func (n networkRouteTableWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkRouteTableLookupByName,
	}
}

func (n networkRouteTableWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkRoute,
		azureshared.NetworkSubnet,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route_table
func (n networkRouteTableWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_route_table.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (n networkRouteTableWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/routeTables/read",
	}
}

func (n networkRouteTableWrapper) PredefinedRole() string {
	return "Reader"
}
