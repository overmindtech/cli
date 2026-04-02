package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_flexible_server_administrator_client.go -package=mocks -source=dbforpostgresql-flexible-server-administrator-client.go

// DBforPostgreSQLFlexibleServerAdministratorPager is a type alias for the generic Pager interface with administrator response type.
type DBforPostgreSQLFlexibleServerAdministratorPager = Pager[armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse]

// DBforPostgreSQLFlexibleServerAdministratorClient is an interface for interacting with Azure PostgreSQL Flexible Server Administrators
type DBforPostgreSQLFlexibleServerAdministratorClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerAdministratorPager
	Get(ctx context.Context, resourceGroupName string, serverName string, objectID string) (armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientGetResponse, error)
}

type dbforPostgreSQLFlexibleServerAdministratorClient struct {
	client *armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClient
}

func (a *dbforPostgreSQLFlexibleServerAdministratorClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerAdministratorPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *dbforPostgreSQLFlexibleServerAdministratorClient) Get(ctx context.Context, resourceGroupName string, serverName string, objectID string) (armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, objectID, nil)
}

// NewDBforPostgreSQLFlexibleServerAdministratorClient creates a new DBforPostgreSQLFlexibleServerAdministratorClient from the Azure SDK client
func NewDBforPostgreSQLFlexibleServerAdministratorClient(client *armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClient) DBforPostgreSQLFlexibleServerAdministratorClient {
	return &dbforPostgreSQLFlexibleServerAdministratorClient{client: client}
}
