package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_databases_client.go -package=mocks -source=sql-databases-client.go

// SqlDatabasesPager is a type alias for the generic Pager interface with sql database response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type SqlDatabasesPager = Pager[armsql.DatabasesClientListByServerResponse]

// SqlDatabasesClient is an interface for interacting with Azure sql databases
type SqlDatabasesClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlDatabasesPager
	Get(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (armsql.DatabasesClientGetResponse, error)
}

type sqlDatabasesClient struct {
	client *armsql.DatabasesClient
}

func (a *sqlDatabasesClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlDatabasesPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlDatabasesClient) Get(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (armsql.DatabasesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, databaseName, nil)
}

// NewSqlDatabasesClient creates a new SqlDatabasesClient from the Azure SDK client
func NewSqlDatabasesClient(client *armsql.DatabasesClient) SqlDatabasesClient {
	return &sqlDatabasesClient{client: client}
}
