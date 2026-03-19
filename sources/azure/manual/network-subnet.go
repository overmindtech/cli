package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var NetworkSubnetLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkSubnet)

type networkSubnetWrapper struct {
	client clients.SubnetsClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkSubnet creates a new networkSubnetWrapper instance (SearchableWrapper: child of virtual network).
func NewNetworkSubnet(client clients.SubnetsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkSubnetWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkSubnet,
		),
	}
}

func (n networkSubnetWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: virtualNetworkName and subnetName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	virtualNetworkName := queryParts[0]
	subnetName := queryParts[1]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, virtualNetworkName, subnetName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureSubnetToSDPItem(&resp.Subnet, virtualNetworkName, subnetName, scope)
}

func (n networkSubnetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkVirtualNetworkLookupByName,
		NetworkSubnetLookupByUniqueAttr,
	}
}

func (n networkSubnetWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: virtualNetworkName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	virtualNetworkName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, virtualNetworkName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, subnet := range page.Value {
			if subnet == nil || subnet.Name == nil {
				continue
			}
			item, sdpErr := n.azureSubnetToSDPItem(subnet, virtualNetworkName, *subnet.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkSubnetWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: virtualNetworkName"), scope, n.Type()))
		return
	}
	virtualNetworkName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, virtualNetworkName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, subnet := range page.Value {
			if subnet == nil || subnet.Name == nil {
				continue
			}
			item, sdpErr := n.azureSubnetToSDPItem(subnet, virtualNetworkName, *subnet.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkSubnetWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkVirtualNetworkLookupByName,
		},
	}
}

func (n networkSubnetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkVirtualNetwork:        true,
		azureshared.NetworkNetworkSecurityGroup:  true,
		azureshared.NetworkRouteTable:            true,
		azureshared.NetworkNatGateway:            true,
		azureshared.NetworkPrivateEndpoint:       true,
		azureshared.NetworkServiceEndpointPolicy: true,
		azureshared.NetworkIpAllocation:          true,
		azureshared.NetworkNetworkInterface:      true,
		azureshared.NetworkApplicationGateway:    true,
	}
}

func (n networkSubnetWrapper) azureSubnetToSDPItem(subnet *armnetwork.Subnet, virtualNetworkName, subnetName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(subnet, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(virtualNetworkName, subnetName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkSubnet.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to parent Virtual Network
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkVirtualNetwork.String(),
			Method: sdp.QueryMethod_GET,
			Query:  virtualNetworkName,
			Scope:  scope,
		},
	})

	// Link to Network Security Group from subnet
	if subnet.Properties != nil && subnet.Properties.NetworkSecurityGroup != nil && subnet.Properties.NetworkSecurityGroup.ID != nil {
		nsgID := *subnet.Properties.NetworkSecurityGroup.ID
		nsgName := azureshared.ExtractResourceName(nsgID)
		if nsgName != "" {
			linkScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(nsgID); extractedScope != "" {
				linkScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkNetworkSecurityGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  nsgName,
					Scope:  linkScope,
				},
			})
		}
	}

	// Link to Route Table from subnet
	if subnet.Properties != nil && subnet.Properties.RouteTable != nil && subnet.Properties.RouteTable.ID != nil {
		routeTableID := *subnet.Properties.RouteTable.ID
		routeTableName := azureshared.ExtractResourceName(routeTableID)
		if routeTableName != "" {
			linkScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(routeTableID); extractedScope != "" {
				linkScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkRouteTable.String(),
					Method: sdp.QueryMethod_GET,
					Query:  routeTableName,
					Scope:  linkScope,
				},
			})
		}
	}

	// Link to NAT Gateway from subnet
	if subnet.Properties != nil && subnet.Properties.NatGateway != nil && subnet.Properties.NatGateway.ID != nil {
		natGatewayID := *subnet.Properties.NatGateway.ID
		natGatewayName := azureshared.ExtractResourceName(natGatewayID)
		if natGatewayName != "" {
			linkScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(natGatewayID); extractedScope != "" {
				linkScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkNatGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  natGatewayName,
					Scope:  linkScope,
				},
			})
		}
	}

	// Link to Private Endpoints from subnet (read-only references)
	if subnet.Properties != nil && subnet.Properties.PrivateEndpoints != nil {
		for _, privateEndpoint := range subnet.Properties.PrivateEndpoints {
			if privateEndpoint != nil && privateEndpoint.ID != nil {
				privateEndpointID := *privateEndpoint.ID
				privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
				if privateEndpointName != "" {
					linkScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
						linkScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  privateEndpointName,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Link to Service Endpoint Policies from subnet
	if subnet.Properties != nil && subnet.Properties.ServiceEndpointPolicies != nil {
		for _, policy := range subnet.Properties.ServiceEndpointPolicies {
			if policy != nil && policy.ID != nil {
				policyID := *policy.ID
				policyName := azureshared.ExtractResourceName(policyID)
				if policyName != "" {
					linkScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(policyID); extractedScope != "" {
						linkScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkServiceEndpointPolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  policyName,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Link to IP Allocations from subnet (references that use this subnet)
	if subnet.Properties != nil && subnet.Properties.IPAllocations != nil {
		for _, ipAlloc := range subnet.Properties.IPAllocations {
			if ipAlloc != nil && ipAlloc.ID != nil {
				ipAllocID := *ipAlloc.ID
				ipAllocName := azureshared.ExtractResourceName(ipAllocID)
				if ipAllocName != "" {
					linkScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(ipAllocID); extractedScope != "" {
						linkScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkIpAllocation.String(),
							Method: sdp.QueryMethod_GET,
							Query:  ipAllocName,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Link to Network Interfaces that have IP configurations in this subnet (read-only references)
	if subnet.Properties != nil && subnet.Properties.IPConfigurations != nil {
		for _, ipConfig := range subnet.Properties.IPConfigurations {
			if ipConfig != nil && ipConfig.ID != nil {
				ipConfigID := *ipConfig.ID
				// Format: .../networkInterfaces/{nicName}/ipConfigurations/{ipConfigName}
				if strings.Contains(ipConfigID, "/networkInterfaces/") {
					nicNames := azureshared.ExtractPathParamsFromResourceID(ipConfigID, []string{"networkInterfaces"})
					if len(nicNames) > 0 && nicNames[0] != "" {
						linkScope := azureshared.ExtractScopeFromResourceID(ipConfigID)
						if linkScope == "" {
							linkScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkNetworkInterface.String(),
								Method: sdp.QueryMethod_GET,
								Query:  nicNames[0],
								Scope:  linkScope,
							},
						})
					}
				}
			}
		}
	}

	// Link to Application Gateways that have gateway IP configurations in this subnet (read-only references)
	if subnet.Properties != nil && subnet.Properties.ApplicationGatewayIPConfigurations != nil {
		for _, agIPConfig := range subnet.Properties.ApplicationGatewayIPConfigurations {
			if agIPConfig != nil && agIPConfig.ID != nil {
				agIPConfigID := *agIPConfig.ID
				// Format: .../applicationGateways/{agName}/applicationGatewayIPConfigurations/...
				agNames := azureshared.ExtractPathParamsFromResourceID(agIPConfigID, []string{"applicationGateways"})
				if len(agNames) > 0 && agNames[0] != "" {
					linkScope := azureshared.ExtractScopeFromResourceID(agIPConfigID)
					if linkScope == "" {
						linkScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkApplicationGateway.String(),
							Method: sdp.QueryMethod_GET,
							Query:  agNames[0],
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Link to external resources referenced by ResourceNavigationLinks (e.g. SQL Managed Instance)
	if subnet.Properties != nil && subnet.Properties.ResourceNavigationLinks != nil {
		for _, rnl := range subnet.Properties.ResourceNavigationLinks {
			if rnl != nil && rnl.Properties != nil && rnl.Properties.Link != nil {
				linkID := *rnl.Properties.Link
				resourceName := azureshared.ExtractResourceName(linkID)
				if resourceName != "" {
					linkScope := azureshared.ExtractScopeFromResourceID(linkID)
					if linkScope == "" {
						linkScope = scope
					}
					itemType := azureshared.ItemTypeFromLinkedResourceID(linkID)
					if itemType == "" {
						itemType = "azure-resource"
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   itemType,
							Method: sdp.QueryMethod_GET,
							Query:  resourceName,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Link to external resources referenced by ServiceAssociationLinks (e.g. App Service Environment)
	if subnet.Properties != nil && subnet.Properties.ServiceAssociationLinks != nil {
		for _, sal := range subnet.Properties.ServiceAssociationLinks {
			if sal != nil && sal.Properties != nil && sal.Properties.Link != nil {
				linkID := *sal.Properties.Link
				resourceName := azureshared.ExtractResourceName(linkID)
				if resourceName != "" {
					linkScope := azureshared.ExtractScopeFromResourceID(linkID)
					if linkScope == "" {
						linkScope = scope
					}
					itemType := azureshared.ItemTypeFromLinkedResourceID(linkID)
					if itemType == "" {
						itemType = "azure-resource"
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   itemType,
							Method: sdp.QueryMethod_GET,
							Query:  resourceName,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkSubnetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_subnet.id",
		},
	}
}

func (n networkSubnetWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/virtualNetworks/subnets/read",
	}
}

func (n networkSubnetWrapper) PredefinedRole() string {
	return "Reader"
}
