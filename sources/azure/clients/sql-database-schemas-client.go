package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_database_schemas_client.go -package=mocks -source=sql-database-schemas-client.go

// SqlDatabaseSchemasPager is a type alias for the generic Pager interface with database schema response type.
type SqlDatabaseSchemasPager = Pager[armsql.DatabaseSchemasClientListByDatabaseResponse]

// SqlDatabaseSchemasClient is an interface for interacting with Azure SQL database schemas
type SqlDatabaseSchemasClient interface {
	Get(ctx context.Context, resourceGroupName, serverName, databaseName, schemaName string) (armsql.DatabaseSchemasClientGetResponse, error)
	ListByDatabase(ctx context.Context, resourceGroupName, serverName, databaseName string) SqlDatabaseSchemasPager
}

type sqlDatabaseSchemasClient struct {
	client *armsql.DatabaseSchemasClient
}

func (c *sqlDatabaseSchemasClient) Get(ctx context.Context, resourceGroupName, serverName, databaseName, schemaName string) (armsql.DatabaseSchemasClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, serverName, databaseName, schemaName, nil)
}

func (c *sqlDatabaseSchemasClient) ListByDatabase(ctx context.Context, resourceGroupName, serverName, databaseName string) SqlDatabaseSchemasPager {
	return c.client.NewListByDatabasePager(resourceGroupName, serverName, databaseName, nil)
}

// NewSqlDatabaseSchemasClient creates a new SqlDatabaseSchemasClient from the Azure SDK client
func NewSqlDatabaseSchemasClient(client *armsql.DatabaseSchemasClient) SqlDatabaseSchemasClient {
	return &sqlDatabaseSchemasClient{client: client}
}
