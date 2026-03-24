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
)

var DBforPostgreSQLFlexibleServerConfigurationLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServerConfiguration)

type dbforPostgreSQLFlexibleServerConfigurationWrapper struct {
	client clients.PostgreSQLConfigurationsClient

	*azureshared.MultiResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServerConfiguration(client clients.PostgreSQLConfigurationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &dbforPostgreSQLFlexibleServerConfigurationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServerConfiguration,
		),
	}
}

// Get retrieves a single configuration by server name and configuration name.
// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/configurations/get?view=rest-postgresql-2025-08-01
func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and configurationName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	serverName := queryParts[0]
	configurationName := queryParts[1]

	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	if configurationName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "configurationName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, serverName, configurationName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureConfigurationToSDPItem(&resp.Configuration, serverName, scope)
}

// Search lists all configurations for a given server.
// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/configurations/list-by-server?view=rest-postgresql-2025-08-01
func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	serverName := queryParts[0]

	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	pager := c.client.NewListByServerPager(rgScope.ResourceGroup, serverName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}

		for _, configuration := range page.Value {
			if configuration.Name == nil {
				continue
			}

			item, sdpErr := c.azureConfigurationToSDPItem(configuration, serverName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// SearchStream streams configurations for a given server.
func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, c.Type()))
		return
	}
	serverName := queryParts[0]

	if serverName == "" {
		stream.SendError(azureshared.QueryError(errors.New("serverName cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}

	pager := c.client.NewListByServerPager(rgScope.ResourceGroup, serverName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, configuration := range page.Value {
			if configuration.Name == nil {
				continue
			}

			item, sdpErr := c.azureConfigurationToSDPItem(configuration, serverName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) azureConfigurationToSDPItem(configuration *armpostgresqlflexibleservers.Configuration, serverName, scope string) (*sdp.Item, *sdp.QueryError) {
	if configuration.Name == nil {
		return nil, azureshared.QueryError(errors.New("configuration name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(configuration)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	configurationName := *configuration.Name

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, configurationName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            c.Type(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link back to parent Flexible Server
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  serverName,
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLFlexibleServerConfigurationLookupByName,
	}
}

func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.DBforPostgreSQLFlexibleServer,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/databases#microsoftdbforpostgresql
func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/configurations/read",
	}
}

func (c dbforPostgreSQLFlexibleServerConfigurationWrapper) PredefinedRole() string {
	return "Reader"
}
