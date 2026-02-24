package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var SQLServerFirewallRuleLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServerFirewallRule)

type sqlServerFirewallRuleWrapper struct {
	client clients.SqlServerFirewallRuleClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlServerFirewallRule(client clients.SqlServerFirewallRuleClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlServerFirewallRuleWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServerFirewallRule,
		),
	}
}

func (s sqlServerFirewallRuleWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and firewallRuleName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	firewallRuleName := queryParts[1]
	if firewallRuleName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "firewallRuleName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, firewallRuleName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureSqlServerFirewallRuleToSDPItem(&resp.FirewallRule, serverName, firewallRuleName, scope)
}

func (s sqlServerFirewallRuleWrapper) azureSqlServerFirewallRuleToSDPItem(rule *armsql.FirewallRule, serverName, firewallRuleName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(rule, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, firewallRuleName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServerFirewallRule.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            nil, // FirewallRule has no Tags in the Azure SDK
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

	// Link to stdlib IP items for StartIPAddress and EndIPAddress (global scope, GET)
	if rule.Properties != nil {
		if rule.Properties.StartIPAddress != nil && *rule.Properties.StartIPAddress != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  *rule.Properties.StartIPAddress,
					Scope:  "global",
				},
			})
		}
		if rule.Properties.EndIPAddress != nil && *rule.Properties.EndIPAddress != "" && (rule.Properties.StartIPAddress == nil || *rule.Properties.EndIPAddress != *rule.Properties.StartIPAddress) {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  *rule.Properties.EndIPAddress,
					Scope:  "global",
				},
			})
		}
	}

	return sdpItem, nil
}

func (s sqlServerFirewallRuleWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLServerFirewallRuleLookupByName,
	}
}

func (s sqlServerFirewallRuleWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
			item, sdpErr := s.azureSqlServerFirewallRuleToSDPItem(rule, serverName, *rule.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlServerFirewallRuleWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
			item, sdpErr := s.azureSqlServerFirewallRuleToSDPItem(rule, serverName, *rule.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlServerFirewallRuleWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (s sqlServerFirewallRuleWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLServer: true,
		stdlib.NetworkIP:      true,
	}
}

func (s sqlServerFirewallRuleWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_mssql_firewall_rule.id",
		},
	}
}

func (s sqlServerFirewallRuleWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/firewallRules/read",
	}
}

func (s sqlServerFirewallRuleWrapper) PredefinedRole() string {
	return "Reader"
}
