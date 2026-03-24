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
)

var SQLDatabaseSchemaLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLDatabaseSchema)

type sqlDatabaseSchemaWrapper struct {
	client clients.SqlDatabaseSchemasClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlDatabaseSchema(client clients.SqlDatabaseSchemasClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlDatabaseSchemaWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLDatabaseSchema,
		),
	}
}

// Get retrieves a specific database schema by serverName, databaseName, and schemaName
// ref: https://learn.microsoft.com/en-us/rest/api/sql/database-schemas/get
func (s sqlDatabaseSchemaWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 3 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 3 query parts: serverName, databaseName, and schemaName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]
	schemaName := queryParts[2]

	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if databaseName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "databaseName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if schemaName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "schemaName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, databaseName, schemaName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureDatabaseSchemaToSDPItem(&resp.DatabaseSchema, serverName, databaseName, schemaName, scope)
}

func (s sqlDatabaseSchemaWrapper) azureDatabaseSchemaToSDPItem(schema *armsql.DatabaseSchema, serverName, databaseName, schemaName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(schema)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, databaseName, schemaName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLDatabaseSchema.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to parent SQL Database
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.SQLDatabase.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(serverName, databaseName),
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (s sqlDatabaseSchemaWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLDatabaseLookupByName,
		SQLDatabaseSchemaLookupByName,
	}
}

// Search lists all database schemas for a given serverName and databaseName
// ref: https://learn.microsoft.com/en-us/rest/api/sql/database-schemas/list-by-database
func (s sqlDatabaseSchemaWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 2 query parts: serverName and databaseName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]

	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName cannot be empty"), scope, s.Type())
	}
	if databaseName == "" {
		return nil, azureshared.QueryError(errors.New("databaseName cannot be empty"), scope, s.Type())
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByDatabase(ctx, rgScope.ResourceGroup, serverName, databaseName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}
		for _, schema := range page.Value {
			if schema.Name == nil {
				continue
			}
			item, sdpErr := s.azureDatabaseSchemaToSDPItem(schema, serverName, databaseName, *schema.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlDatabaseSchemaWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 2 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 2 query parts: serverName and databaseName"), scope, s.Type()))
		return
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]

	if serverName == "" {
		stream.SendError(azureshared.QueryError(errors.New("serverName cannot be empty"), scope, s.Type()))
		return
	}
	if databaseName == "" {
		stream.SendError(azureshared.QueryError(errors.New("databaseName cannot be empty"), scope, s.Type()))
		return
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByDatabase(ctx, rgScope.ResourceGroup, serverName, databaseName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, schema := range page.Value {
			if schema.Name == nil {
				continue
			}
			item, sdpErr := s.azureDatabaseSchemaToSDPItem(schema, serverName, databaseName, *schema.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlDatabaseSchemaWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
			SQLDatabaseLookupByName,
		},
	}
}

func (s sqlDatabaseSchemaWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLDatabase: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftsql
func (s sqlDatabaseSchemaWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/databases/schemas/read",
	}
}

func (s sqlDatabaseSchemaWrapper) PredefinedRole() string {
	return "Reader"
}
