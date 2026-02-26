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

var NetworkSecurityRuleLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkSecurityRule)

type networkSecurityRuleWrapper struct {
	client clients.SecurityRulesClient
	*azureshared.MultiResourceGroupBase
}

// NewNetworkSecurityRule creates a new networkSecurityRuleWrapper instance (SearchableWrapper: child of network security group).
func NewNetworkSecurityRule(client clients.SecurityRulesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkSecurityRuleWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkSecurityRule,
		),
	}
}

func (n networkSecurityRuleWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: networkSecurityGroupName and securityRuleName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	nsgName := queryParts[0]
	ruleName := queryParts[1]
	if ruleName == "" {
		return nil, azureshared.QueryError(errors.New("security rule name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, nsgName, ruleName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureSecurityRuleToSDPItem(&resp.SecurityRule, nsgName, ruleName, scope)
}

func (n networkSecurityRuleWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNetworkSecurityGroupLookupByName,
		NetworkSecurityRuleLookupByUniqueAttr,
	}
}

func (n networkSecurityRuleWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: networkSecurityGroupName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	nsgName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nsgName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, rule := range page.Value {
			if rule == nil || rule.Name == nil {
				continue
			}
			item, sdpErr := n.azureSecurityRuleToSDPItem(rule, nsgName, *rule.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkSecurityRuleWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: networkSecurityGroupName"), scope, n.Type()))
		return
	}
	nsgName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nsgName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, rule := range page.Value {
			if rule == nil || rule.Name == nil {
				continue
			}
			item, sdpErr := n.azureSecurityRuleToSDPItem(rule, nsgName, *rule.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkSecurityRuleWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{NetworkNetworkSecurityGroupLookupByName},
	}
}

func (n networkSecurityRuleWrapper) azureSecurityRuleToSDPItem(rule *armnetwork.SecurityRule, nsgName, ruleName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(rule, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(nsgName, ruleName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkSecurityRule.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to parent Network Security Group
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkNetworkSecurityGroup.String(),
			Method: sdp.QueryMethod_GET,
			Query:  nsgName,
			Scope:  scope,
		},
	})

	if rule.Properties != nil {
		// Link to SourceApplicationSecurityGroups
		if rule.Properties.SourceApplicationSecurityGroups != nil {
			for _, asgRef := range rule.Properties.SourceApplicationSecurityGroups {
				if asgRef != nil && asgRef.ID != nil {
					asgName := azureshared.ExtractResourceName(*asgRef.ID)
					if asgName != "" {
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
							linkScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkApplicationSecurityGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  asgName,
								Scope:  linkScope,
							},
						})
					}
				}
			}
		}

		// Link to DestinationApplicationSecurityGroups
		if rule.Properties.DestinationApplicationSecurityGroups != nil {
			for _, asgRef := range rule.Properties.DestinationApplicationSecurityGroups {
				if asgRef != nil && asgRef.ID != nil {
					asgName := azureshared.ExtractResourceName(*asgRef.ID)
					if asgName != "" {
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
							linkScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkApplicationSecurityGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  asgName,
								Scope:  linkScope,
							},
						})
					}
				}
			}
		}

		// Link to stdlib.NetworkIP for source/destination address prefixes when they are IPs or CIDRs
		if rule.Properties.SourceAddressPrefix != nil {
			appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *rule.Properties.SourceAddressPrefix)
		}
		for _, p := range rule.Properties.SourceAddressPrefixes {
			if p != nil {
				appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
			}
		}
		if rule.Properties.DestinationAddressPrefix != nil {
			appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *rule.Properties.DestinationAddressPrefix)
		}
		for _, p := range rule.Properties.DestinationAddressPrefixes {
			if p != nil {
				appendIPOrCIDRLinkIfValid(&sdpItem.LinkedItemQueries, *p)
			}
		}
	}

	return sdpItem, nil
}

func (n networkSecurityRuleWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.NetworkApplicationSecurityGroup,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_rule
func (n networkSecurityRuleWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_network_security_rule.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions-reference#microsoftnetwork
func (n networkSecurityRuleWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkSecurityGroups/securityRules/read",
	}
}

func (n networkSecurityRuleWrapper) PredefinedRole() string {
	return "Reader"
}
