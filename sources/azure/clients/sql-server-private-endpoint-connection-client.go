package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_server_private_endpoint_connection_client.go -package=mocks -source=sql-server-private-endpoint-connection-client.go

// SQLServerPrivateEndpointConnectionsPager is a type alias for the generic Pager interface with SQL server private endpoint connection list response type.
type SQLServerPrivateEndpointConnectionsPager = Pager[armsql.PrivateEndpointConnectionsClientListByServerResponse]

// SQLServerPrivateEndpointConnectionsClient is an interface for interacting with Azure SQL server private endpoint connections.
type SQLServerPrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, serverName string, privateEndpointConnectionName string) (armsql.PrivateEndpointConnectionsClientGetResponse, error)
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SQLServerPrivateEndpointConnectionsPager
}

type sqlServerPrivateEndpointConnectionsClient struct {
	client *armsql.PrivateEndpointConnectionsClient
}

func (c *sqlServerPrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, serverName string, privateEndpointConnectionName string) (armsql.PrivateEndpointConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, serverName, privateEndpointConnectionName, nil)
}

func (c *sqlServerPrivateEndpointConnectionsClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SQLServerPrivateEndpointConnectionsPager {
	return c.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

// NewSQLServerPrivateEndpointConnectionsClient creates a new SQLServerPrivateEndpointConnectionsClient from the Azure SDK client.
func NewSQLServerPrivateEndpointConnectionsClient(client *armsql.PrivateEndpointConnectionsClient) SQLServerPrivateEndpointConnectionsClient {
	return &sqlServerPrivateEndpointConnectionsClient{client: client}
}
