package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var SQLServerVirtualNetworkRuleLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServerVirtualNetworkRule)

type sqlServerVirtualNetworkRuleWrapper struct {
	client clients.SqlServerVirtualNetworkRuleClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlServerVirtualNetworkRule(client clients.SqlServerVirtualNetworkRuleClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlServerVirtualNetworkRuleWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServerVirtualNetworkRule,
		),
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and virtualNetworkRuleName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	ruleName := queryParts[1]
	if ruleName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "virtualNetworkRuleName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, ruleName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureSqlServerVirtualNetworkRuleToSDPItem(&resp.VirtualNetworkRule, serverName, ruleName, scope)
}

func (s sqlServerVirtualNetworkRuleWrapper) azureSqlServerVirtualNetworkRuleToSDPItem(rule *armsql.VirtualNetworkRule, serverName, ruleName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(rule, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, ruleName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServerVirtualNetworkRule.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            nil, // VirtualNetworkRule has no Tags in the Azure SDK
	}

	// Link to parent SQL Server (from resource ID or known server name)
	if rule.ID != nil {
		extractedServerName := azureshared.ExtractSQLServerNameFromDatabaseID(*rule.ID)
		if extractedServerName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  extractedServerName,
					Scope:  scope,
				},
			})
		}
	} else {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServer.String(),
				Method: sdp.QueryMethod_GET,
				Query:  serverName,
				Scope:  scope,
			},
		})
	}

	// Link to Virtual Network and Subnet when VirtualNetworkSubnetID is set
	// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}
	if rule.Properties != nil && rule.Properties.VirtualNetworkSubnetID != nil {
		subnetID := *rule.Properties.VirtualNetworkSubnetID
		scopeParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"subscriptions", "resourceGroups"})
		subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
		if len(scopeParams) >= 2 && len(subnetParams) >= 2 {
			subscriptionID := scopeParams[0]
			resourceGroupName := scopeParams[1]
			vnetName := subnetParams[0]
			subnetName := subnetParams[1]
			subnetScope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
			// Link to Virtual Network
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vnetName,
					Scope:  subnetScope,
				},
			})
			// Link to Subnet
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkSubnet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(vnetName, subnetName),
					Scope:  subnetScope,
				},
			})
		}
	}

	return sdpItem, nil
}

func (s sqlServerVirtualNetworkRuleWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLServerVirtualNetworkRuleLookupByName,
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, rule := range page.Value {
			if rule.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlServerVirtualNetworkRuleToSDPItem(rule, serverName, *rule.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlServerVirtualNetworkRuleWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
		for _, rule := range page.Value {
			if rule.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlServerVirtualNetworkRuleToSDPItem(rule, serverName, *rule.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLServer:             true,
		azureshared.NetworkSubnet:         true,
		azureshared.NetworkVirtualNetwork: true,
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_mssql_virtual_network_rule.id",
		},
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/virtualNetworkRules/read",
	}
}

func (s sqlServerVirtualNetworkRuleWrapper) PredefinedRole() string {
	return "Reader"
}
