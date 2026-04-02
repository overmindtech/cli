package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_flexible_server_virtual_endpoint_client.go -package=mocks -source=dbforpostgresql-flexible-server-virtual-endpoint-client.go

type DBforPostgreSQLFlexibleServerVirtualEndpointPager = Pager[armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse]

type DBforPostgreSQLFlexibleServerVirtualEndpointClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerVirtualEndpointPager
	Get(ctx context.Context, resourceGroupName string, serverName string, virtualEndpointName string) (armpostgresqlflexibleservers.VirtualEndpointsClientGetResponse, error)
}

type dbforPostgreSQLFlexibleServerVirtualEndpointClient struct {
	client *armpostgresqlflexibleservers.VirtualEndpointsClient
}

func (a *dbforPostgreSQLFlexibleServerVirtualEndpointClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerVirtualEndpointPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *dbforPostgreSQLFlexibleServerVirtualEndpointClient) Get(ctx context.Context, resourceGroupName string, serverName string, virtualEndpointName string) (armpostgresqlflexibleservers.VirtualEndpointsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, virtualEndpointName, nil)
}

func NewDBforPostgreSQLFlexibleServerVirtualEndpointClient(client *armpostgresqlflexibleservers.VirtualEndpointsClient) DBforPostgreSQLFlexibleServerVirtualEndpointClient {
	return &dbforPostgreSQLFlexibleServerVirtualEndpointClient{client: client}
}
