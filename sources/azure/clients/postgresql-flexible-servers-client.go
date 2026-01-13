package clients

import (
	"context"

	armpostgresqlflexibleservers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_postgresql_flexible_servers_client.go -package=mocks -source=postgresql-flexible-servers-client.go

// PostgreSQLFlexibleServersPager is a type alias for the generic Pager interface with postgresql flexible server response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type PostgreSQLFlexibleServersPager = Pager[armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse]

// PostgreSQLFlexibleServersClient is an interface for interacting with Azure postgresql flexible servers
type PostgreSQLFlexibleServersClient interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armpostgresqlflexibleservers.ServersClientListByResourceGroupOptions) PostgreSQLFlexibleServersPager
	Get(ctx context.Context, resourceGroupName string, serverName string, options *armpostgresqlflexibleservers.ServersClientGetOptions) (armpostgresqlflexibleservers.ServersClientGetResponse, error)
}

type postgresqlFlexibleServersClient struct {
	client *armpostgresqlflexibleservers.ServersClient
}

func (a *postgresqlFlexibleServersClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armpostgresqlflexibleservers.ServersClientListByResourceGroupOptions) PostgreSQLFlexibleServersPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *postgresqlFlexibleServersClient) Get(ctx context.Context, resourceGroupName string, serverName string, options *armpostgresqlflexibleservers.ServersClientGetOptions) (armpostgresqlflexibleservers.ServersClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, options)
}

// NewPostgreSQLFlexibleServersClient creates a new PostgreSQLFlexibleServersClient from the Azure SDK client
func NewPostgreSQLFlexibleServersClient(client *armpostgresqlflexibleservers.ServersClient) PostgreSQLFlexibleServersClient {
	return &postgresqlFlexibleServersClient{client: client}
}
