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

var SQLElasticPoolLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLElasticPool)

type sqlElasticPoolWrapper struct {
	client clients.SqlElasticPoolClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlElasticPool(client clients.SqlElasticPoolClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlElasticPoolWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLElasticPool,
		),
	}
}

func (s sqlElasticPoolWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and elasticPoolName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	elasticPoolName := queryParts[1]
	if elasticPoolName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "elasticPoolName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, elasticPoolName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureSqlElasticPoolToSDPItem(&resp.ElasticPool, serverName, elasticPoolName, scope)
}

func (s sqlElasticPoolWrapper) azureSqlElasticPoolToSDPItem(pool *armsql.ElasticPool, serverName, elasticPoolName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(pool, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, elasticPoolName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLElasticPool.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(pool.Tags),
	}

	// Link to parent SQL Server (from resource ID or known server name)
	if pool.ID != nil {
		extractedServerName := azureshared.ExtractPathParamsFromResourceID(*pool.ID, []string{"servers"})
		if len(extractedServerName) >= 1 && extractedServerName[0] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  extractedServerName[0],
					Scope:  scope,
				},
			})
		}
	}
	if len(sdpItem.GetLinkedItemQueries()) == 0 {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServer.String(),
				Method: sdp.QueryMethod_GET,
				Query:  serverName,
				Scope:  scope,
			},
		})
	}

	// Link to Maintenance Configuration when set
	if pool.Properties != nil && pool.Properties.MaintenanceConfigurationID != nil && *pool.Properties.MaintenanceConfigurationID != "" {
		configName := azureshared.ExtractResourceName(*pool.Properties.MaintenanceConfigurationID)
		if configName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(*pool.Properties.MaintenanceConfigurationID)
			if linkedScope == "" && strings.Contains(*pool.Properties.MaintenanceConfigurationID, "publicMaintenanceConfigurations") {
				linkedScope = azureshared.ExtractSubscriptionIDFromResourceID(*pool.Properties.MaintenanceConfigurationID)
			}
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.MaintenanceMaintenanceConfiguration.String(),
					Method: sdp.QueryMethod_GET,
					Query:  configName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Link to SQL Databases (child resource; list by server returns all databases; those in this pool reference this pool via ElasticPoolID)
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.SQLDatabase.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  serverName,
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (s sqlElasticPoolWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLElasticPoolLookupByName,
	}
}

func (s sqlElasticPoolWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, pool := range page.Value {
			if pool.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlElasticPoolToSDPItem(pool, serverName, *pool.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlElasticPoolWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
		for _, pool := range page.Value {
			if pool.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlElasticPoolToSDPItem(pool, serverName, *pool.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlElasticPoolWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (s sqlElasticPoolWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLServer:                           true,
		azureshared.SQLDatabase:                         true,
		azureshared.MaintenanceMaintenanceConfiguration: true,
	}
}

func (s sqlElasticPoolWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_mssql_elasticpool.id",
		},
	}
}

func (s sqlElasticPoolWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/elasticPools/read",
	}
}

func (s sqlElasticPoolWrapper) PredefinedRole() string {
	return "Reader"
}
