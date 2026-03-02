package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_documentdb_private_endpoint_connection_client.go -package=mocks -source=documentdb-private-endpoint-connection-client.go

// DocumentDBPrivateEndpointConnectionsPager is a type alias for the generic Pager interface with Cosmos DB private endpoint connection list response type.
type DocumentDBPrivateEndpointConnectionsPager = Pager[armcosmos.PrivateEndpointConnectionsClientListByDatabaseAccountResponse]

// DocumentDBPrivateEndpointConnectionsClient is an interface for interacting with Azure Cosmos DB (DocumentDB) database account private endpoint connections.
type DocumentDBPrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armcosmos.PrivateEndpointConnectionsClientGetResponse, error)
	ListByDatabaseAccount(ctx context.Context, resourceGroupName string, accountName string) DocumentDBPrivateEndpointConnectionsPager
}

type documentDBPrivateEndpointConnectionsClient struct {
	client *armcosmos.PrivateEndpointConnectionsClient
}

func (c *documentDBPrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armcosmos.PrivateEndpointConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, privateEndpointConnectionName, nil)
}

func (c *documentDBPrivateEndpointConnectionsClient) ListByDatabaseAccount(ctx context.Context, resourceGroupName string, accountName string) DocumentDBPrivateEndpointConnectionsPager {
	return c.client.NewListByDatabaseAccountPager(resourceGroupName, accountName, nil)
}

// NewDocumentDBPrivateEndpointConnectionsClient creates a new DocumentDBPrivateEndpointConnectionsClient from the Azure SDK client.
func NewDocumentDBPrivateEndpointConnectionsClient(client *armcosmos.PrivateEndpointConnectionsClient) DocumentDBPrivateEndpointConnectionsClient {
	return &documentDBPrivateEndpointConnectionsClient{client: client}
}
