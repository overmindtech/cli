package manual

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkNetworkSecurityGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkNetworkSecurityGroup)

// appendIPOrCIDRLinkIfValid appends a linked item query to stdlib.NetworkIP when the prefix is an IP address or CIDR (not a service tag like VirtualNetwork, Internet, *).
func appendIPOrCIDRLinkIfValid(queries *[]*sdp.LinkedItemQuery, prefix string) {
	appendLinkIfValid(queries, prefix, []string{"*"}, func(p string) *sdp.LinkedItemQuery {
		if net.ParseIP(p) != nil {
			return networkIPQuery(p)
		}
		if _, _, err := net.ParseCIDR(p); err == nil {
			return networkIPQuery(p)
		}
		return nil
	})
}

type networkNetworkSecurityGroupWrapper struct {
	client clients.NetworkSecurityGroupsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkNetworkSecurityGroup(client clients.NetworkSecurityGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkNetworkSecurityGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkNetworkSecurityGroup,
		),
	}
}

// reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/list?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
func (n networkNetworkSecurityGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.List(ctx, rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}
		for _, networkSecurityGroup := range page.Value {
			if networkSecurityGroup.Name == nil {
				continue
			}
			item, sdpErr := n.azureNetworkSecurityGroupToSDPItem(networkSecurityGroup)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkNetworkSecurityGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.List(ctx, rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, n.DefaultScope(), n.Type()))
			return
		}
		for _, networkSecurityGroup := range page.Value {
			if networkSecurityGroup.Name == nil {
				continue
			}
			item, sdpErr := n.azureNetworkSecurityGroupToSDPItem(networkSecurityGroup)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/get?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
func (n networkNetworkSecurityGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the network security group name"), n.DefaultScope(), n.Type())
	}
	networkSecurityGroupName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	networkSecurityGroup, err := n.client.Get(ctx, rgScope.ResourceGroup, networkSecurityGroupName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}
	return n.azureNetworkSecurityGroupToSDPItem(&networkSecurityGroup.SecurityGroup)
}

func (n networkNetworkSecurityGroupWrapper) azureNetworkSecurityGroupToSDPItem(networkSecurityGroup *armnetwork.SecurityGroup) (*sdp.Item, *sdp.QueryError) {
	if networkSecurityGroup.Name == nil {
		return nil, azureshared.QueryError(errors.New("network security group name is nil"), n.DefaultScope(), n.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(networkSecurityGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}
	nsgName := *networkSecurityGroup.Name
	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkNetworkSecurityGroup.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(networkSecurityGroup.Tags),
	}

	// Link to SecurityRules (child resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/get
	if networkSecurityGroup.Properties != nil && networkSecurityGroup.Properties.SecurityRules != nil {
		for _, securityRule := range networkSecurityGroup.Properties.SecurityRules {
			if securityRule.Name != nil && *securityRule.Name != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkSecurityRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(nsgName, *securityRule.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Security rule changes affect the NSG's behavior
						Out: false, // NSG changes don't affect individual rules (rules are part of NSG)
					},
				})
			}
		}
	}

	// Link to DefaultSecurityRules (child resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/get#defaultsecurityrules
	if networkSecurityGroup.Properties != nil && networkSecurityGroup.Properties.DefaultSecurityRules != nil {
		for _, defaultSecurityRule := range networkSecurityGroup.Properties.DefaultSecurityRules {
			if defaultSecurityRule.Name != nil && *defaultSecurityRule.Name != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkDefaultSecurityRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(nsgName, *defaultSecurityRule.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Default security rule changes affect the NSG's behavior
						Out: false, // NSG changes don't affect individual default rules (rules are part of NSG)
					},
				})
			}
		}
	}

	// Link to Subnets (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	if networkSecurityGroup.Properties != nil && networkSecurityGroup.Properties.Subnets != nil {
		for _, subnetRef := range networkSecurityGroup.Properties.Subnets {
			if subnetRef.ID != nil {
				// Extract subnet name and virtual network name from the resource ID
				// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
				subnetName := azureshared.ExtractResourceName(*subnetRef.ID)
				if subnetName != "" {
					// Extract virtual network name (second to last segment)
					parts := strings.Split(strings.Trim(*subnetRef.ID, "/"), "/")
					vnetName := ""
					for i, part := range parts {
						if part == "virtualNetworks" && i+1 < len(parts) {
							vnetName = parts[i+1]
							break
						}
					}
					if vnetName != "" {
						scope := n.DefaultScope()
						// Check if subnet is in a different resource group
						if extractedScope := azureshared.ExtractScopeFromResourceID(*subnetRef.ID); extractedScope != "" {
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
								In:  true, // Subnet changes (like deletion) affect the NSG association
								Out: true, // NSG rule changes affect traffic in the subnet
							},
						})
					}
				}
			}
		}
	}

	// Link to NetworkInterfaces (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/get
	if networkSecurityGroup.Properties != nil && networkSecurityGroup.Properties.NetworkInterfaces != nil {
		for _, nicRef := range networkSecurityGroup.Properties.NetworkInterfaces {
			if nicRef.ID != nil {
				nicName := azureshared.ExtractResourceName(*nicRef.ID)
				if nicName != "" {
					scope := n.DefaultScope()
					// Check if network interface is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*nicRef.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  false, // Network interface changes don't affect the NSG
							Out: true,  // NSG rule changes affect traffic on the network interface
						},
					})
				}
			}
		}
	}

	// Link to FlowLogs (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/network-watcher/flow-logs/get
	if networkSecurityGroup.Properties != nil && networkSecurityGroup.Properties.FlowLogs != nil {
		for _, flowLogRef := range networkSecurityGroup.Properties.FlowLogs {
			if flowLogRef != nil && flowLogRef.ID != nil && *flowLogRef.ID != "" {
				flowLogID := *flowLogRef.ID
				params := azureshared.ExtractPathParamsFromResourceID(flowLogID, []string{"networkWatchers", "flowLogs"})
				if len(params) < 2 {
					params = azureshared.ExtractPathParamsFromResourceID(flowLogID, []string{"networkWatchers", "FlowLogs"})
				}
				if len(params) >= 2 {
					networkWatcherName := params[0]
					flowLogName := params[1]
					scope := n.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(flowLogID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkFlowLog.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(networkWatcherName, flowLogName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Flow log config changes affect the NSG's observability
							Out: false, // NSG changes don't affect the flow log resource
						},
					})
				}
			}
		}
	}

	// Link to ApplicationSecurityGroups and IPGroups from SecurityRules
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/application-security-groups/get
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ip-groups/get
	if networkSecurityGroup.Properties != nil {
		// Process SecurityRules
		if networkSecurityGroup.Properties.SecurityRules != nil {
			for _, securityRule := range networkSecurityGroup.Properties.SecurityRules {
				if securityRule.Properties != nil {
					// Link to SourceApplicationSecurityGroups
					if securityRule.Properties.SourceApplicationSecurityGroups != nil {
						for _, asgRef := range securityRule.Properties.SourceApplicationSecurityGroups {
							if asgRef.ID != nil {
								asgName := azureshared.ExtractResourceName(*asgRef.ID)
								if asgName != "" {
									scope := n.DefaultScope()
									// Check if Application Security Group is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
										scope = extractedScope
									}
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkApplicationSecurityGroup.String(),
											Method: sdp.QueryMethod_GET,
											Query:  asgName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // ASG changes affect the security rule's source criteria
											Out: false, // Security rule changes don't affect the ASG
										},
									})
								}
							}
						}
					}

					// Link to DestinationApplicationSecurityGroups
					if securityRule.Properties.DestinationApplicationSecurityGroups != nil {
						for _, asgRef := range securityRule.Properties.DestinationApplicationSecurityGroups {
							if asgRef.ID != nil {
								asgName := azureshared.ExtractResourceName(*asgRef.ID)
								if asgName != "" {
									scope := n.DefaultScope()
									// Check if Application Security Group is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
										scope = extractedScope
									}
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkApplicationSecurityGroup.String(),
											Method: sdp.QueryMethod_GET,
											Query:  asgName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // ASG changes affect the security rule's destination criteria
											Out: false, // Security rule changes don't affect the ASG
										},
									})
								}
							}
						}
					}

					// Link to stdlib.NetworkIP for source/destination address prefixes when they are IPs or CIDRs
					if securityRule.Properties.SourceAddressPrefix != nil {
						appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *securityRule.Properties.SourceAddressPrefix)
					}
					for _, p := range securityRule.Properties.SourceAddressPrefixes {
						if p != nil {
							appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
						}
					}
					if securityRule.Properties.DestinationAddressPrefix != nil {
						appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *securityRule.Properties.DestinationAddressPrefix)
					}
					for _, p := range securityRule.Properties.DestinationAddressPrefixes {
						if p != nil {
							appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
						}
					}
				}
			}
		}

		// Process DefaultSecurityRules (they can also reference ApplicationSecurityGroups and IPGroups)
		if networkSecurityGroup.Properties.DefaultSecurityRules != nil {
			for _, defaultSecurityRule := range networkSecurityGroup.Properties.DefaultSecurityRules {
				if defaultSecurityRule.Properties != nil {
					// Link to SourceApplicationSecurityGroups
					if defaultSecurityRule.Properties.SourceApplicationSecurityGroups != nil {
						for _, asgRef := range defaultSecurityRule.Properties.SourceApplicationSecurityGroups {
							if asgRef.ID != nil {
								asgName := azureshared.ExtractResourceName(*asgRef.ID)
								if asgName != "" {
									scope := n.DefaultScope()
									// Check if Application Security Group is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
										scope = extractedScope
									}
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkApplicationSecurityGroup.String(),
											Method: sdp.QueryMethod_GET,
											Query:  asgName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // ASG changes affect the default security rule's source criteria
											Out: false, // Default security rule changes don't affect the ASG
										},
									})
								}
							}
						}
					}

					// Link to DestinationApplicationSecurityGroups
					if defaultSecurityRule.Properties.DestinationApplicationSecurityGroups != nil {
						for _, asgRef := range defaultSecurityRule.Properties.DestinationApplicationSecurityGroups {
							if asgRef.ID != nil {
								asgName := azureshared.ExtractResourceName(*asgRef.ID)
								if asgName != "" {
									scope := n.DefaultScope()
									// Check if Application Security Group is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
										scope = extractedScope
									}
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkApplicationSecurityGroup.String(),
											Method: sdp.QueryMethod_GET,
											Query:  asgName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // ASG changes affect the default security rule's destination criteria
											Out: false, // Default security rule changes don't affect the ASG
										},
									})
								}
							}
						}
					}

					// Link to stdlib.NetworkIP for source/destination address prefixes when they are IPs or CIDRs
					if defaultSecurityRule.Properties.SourceAddressPrefix != nil {
						appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *defaultSecurityRule.Properties.SourceAddressPrefix)
					}
					for _, p := range defaultSecurityRule.Properties.SourceAddressPrefixes {
						if p != nil {
							appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
						}
					}
					if defaultSecurityRule.Properties.DestinationAddressPrefix != nil {
						appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *defaultSecurityRule.Properties.DestinationAddressPrefix)
					}
					for _, p := range defaultSecurityRule.Properties.DestinationAddressPrefixes {
						if p != nil {
							appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
						}
					}
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkNetworkSecurityGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNetworkSecurityGroupLookupByName,
	}
}

func (n networkNetworkSecurityGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkSecurityRule:             true,
		azureshared.NetworkDefaultSecurityRule:      true,
		azureshared.NetworkSubnet:                   true,
		azureshared.NetworkNetworkInterface:         true,
		azureshared.NetworkFlowLog:                  true,
		azureshared.NetworkApplicationSecurityGroup: true,
		azureshared.NetworkIPGroup:                  true,
		stdlib.NetworkIP:                            true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group
func (n networkNetworkSecurityGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_network_security_group.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (n networkNetworkSecurityGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkSecurityGroups/read",
	}
}

func (n networkNetworkSecurityGroupWrapper) PredefinedRole() string {
	return "Reader"
}
