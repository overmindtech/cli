package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"
)

//go:generate mockgen -destination=../shared/mocks/mock_postgresql_databases_client.go -package=mocks -source=postgresql-databases-client.go

// PostgreSQLDatabasesPager is a type alias for the generic Pager interface with postgresql database response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type PostgreSQLDatabasesPager = Pager[armpostgresqlflexibleservers.DatabasesClientListByServerResponse]

// PostgreSQLDatabasesClient is an interface for interacting with Azure postgresql databases
type PostgreSQLDatabasesClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) PostgreSQLDatabasesPager
	Get(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (armpostgresqlflexibleservers.DatabasesClientGetResponse, error)
}

type postgresqlDatabasesClient struct {
	client *armpostgresqlflexibleservers.DatabasesClient
}

func (a *postgresqlDatabasesClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) PostgreSQLDatabasesPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *postgresqlDatabasesClient) Get(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (armpostgresqlflexibleservers.DatabasesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, databaseName, nil)
}

// NewPostgreSQLDatabasesClient creates a new PostgreSQLDatabasesClient from the Azure SDK client
func NewPostgreSQLDatabasesClient(client *armpostgresqlflexibleservers.DatabasesClient) PostgreSQLDatabasesClient {
	return &postgresqlDatabasesClient{client: client}
}
