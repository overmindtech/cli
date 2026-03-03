package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_flexible_server_private_endpoint_connection_client.go -package=mocks -source=dbforpostgresql-flexible-server-private-endpoint-connection-client.go

// DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager is a type alias for the generic Pager interface with PostgreSQL flexible server private endpoint connection list response type.
type DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager = Pager[armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse]

// DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient is an interface for interacting with Azure DB for PostgreSQL flexible server private endpoint connections.
type DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, serverName string, privateEndpointConnectionName string) (armpostgresqlflexibleservers.PrivateEndpointConnectionsClientGetResponse, error)
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager
}

type dbforpostgresqlFlexibleServerPrivateEndpointConnectionsClient struct {
	client *armpostgresqlflexibleservers.PrivateEndpointConnectionsClient
}

func (c *dbforpostgresqlFlexibleServerPrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, serverName string, privateEndpointConnectionName string) (armpostgresqlflexibleservers.PrivateEndpointConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, serverName, privateEndpointConnectionName, nil)
}

func (c *dbforpostgresqlFlexibleServerPrivateEndpointConnectionsClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager {
	return c.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

// NewDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient creates a new DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient from the Azure SDK client.
func NewDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(client *armpostgresqlflexibleservers.PrivateEndpointConnectionsClient) DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient {
	return &dbforpostgresqlFlexibleServerPrivateEndpointConnectionsClient{client: client}
}
