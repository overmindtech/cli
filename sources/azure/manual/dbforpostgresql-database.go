package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var DBforPostgreSQLDatabaseLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLDatabase)

type dbforPostgreSQLDatabaseWrapper struct {
	client clients.PostgreSQLDatabasesClient

	*azureshared.ResourceGroupBase
}

func NewDBforPostgreSQLDatabase(client clients.PostgreSQLDatabasesClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &dbforPostgreSQLDatabaseWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLDatabase,
		),
	}
}

// reference : https://learn.microsoft.com/en-us/rest/api/postgresql/databases/get?view=rest-postgresql-2025-08-01&tabs=HTTP
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases/{databaseName}?api-version=2025-08-01
func (s dbforPostgreSQLDatabaseWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and databaseName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}
	resp, err := s.client.Get(ctx, resourceGroup, serverName, databaseName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureDBforPostgreSQLDatabaseToSDPItem(&resp.Database, serverName, scope)
}

func (s dbforPostgreSQLDatabaseWrapper) azureDBforPostgreSQLDatabaseToSDPItem(database *armpostgresqlflexibleservers.Database, serverName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(database)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	if database.Name == nil {
		return nil, azureshared.QueryError(fmt.Errorf("database name is nil"), scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, *database.Name))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DBforPostgreSQLDatabase.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to PostgreSQL Flexible Server (parent resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/postgresql/databases/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP
	// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases?api-version=2025-08-01
	//
	// The database is a child resource of the server, so the server is always in the same resource group.
	// We use the serverName that's already available from the query parameters.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  serverName,
			Scope:  scope, // Server is in the same resource group as the database
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Server changes (deletion, configuration, maintenance) directly affect database availability and functionality
			Out: false, // Database changes (schema, data) don't directly affect the server's configuration or operation
		}, // Database depends on server - server is the parent resource that hosts the database
	})

	return sdpItem, nil
}

// reference : https://learn.microsoft.com/en-us/rest/api/postgresql/databases/list-by-server?view=rest-postgresql-2025-08-01&tabs=HTTP#security
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases?api-version=2025-08-01
func (s dbforPostgreSQLDatabaseWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}
	pager := s.client.ListByServer(ctx, resourceGroup, serverName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, database := range page.Value {
			if database.Name == nil {
				continue
			}
			item, sdpErr := s.azureDBforPostgreSQLDatabaseToSDPItem(database, serverName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

// reference: GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases/{databaseName}?api-version=2025-08-01
func (s dbforPostgreSQLDatabaseWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLDatabaseLookupByName,
	}
}

// reference: GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{serverName}/databases?api-version=2025-08-01
func (s dbforPostgreSQLDatabaseWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (s dbforPostgreSQLDatabaseWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.DBforPostgreSQLFlexibleServer: true, // Linked to parent PostgreSQL Flexible Server
	}
}

// reference : https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/postgresql_flexible_server_database
func (s dbforPostgreSQLDatabaseWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_postgresql_flexible_server_database.id",
		},
	}
}

// reference : https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/databases#microsoftdbforpostgresql
func (s dbforPostgreSQLDatabaseWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/databases/read",
	}
}

func (s dbforPostgreSQLDatabaseWrapper) PredefinedRole() string {
	return "Reader"
}
