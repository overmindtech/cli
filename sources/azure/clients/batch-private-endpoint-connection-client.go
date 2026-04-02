package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
)

//go:generate mockgen -destination=../shared/mocks/mock_batch_private_endpoint_connection_client.go -package=mocks -source=batch-private-endpoint-connection-client.go

// BatchPrivateEndpointConnectionPager is a type alias for the generic Pager interface with Batch private endpoint connection list response type.
type BatchPrivateEndpointConnectionPager = Pager[armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse]

// BatchPrivateEndpointConnectionClient is an interface for interacting with Azure Batch private endpoint connections.
type BatchPrivateEndpointConnectionClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armbatch.PrivateEndpointConnectionClientGetResponse, error)
	ListByBatchAccount(ctx context.Context, resourceGroupName string, accountName string) BatchPrivateEndpointConnectionPager
}

type batchPrivateEndpointConnectionClient struct {
	client *armbatch.PrivateEndpointConnectionClient
}

func (c *batchPrivateEndpointConnectionClient) Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armbatch.PrivateEndpointConnectionClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, privateEndpointConnectionName, nil)
}

func (c *batchPrivateEndpointConnectionClient) ListByBatchAccount(ctx context.Context, resourceGroupName string, accountName string) BatchPrivateEndpointConnectionPager {
	return c.client.NewListByBatchAccountPager(resourceGroupName, accountName, nil)
}

// NewBatchPrivateEndpointConnectionClient creates a new BatchPrivateEndpointConnectionClient from the Azure SDK client.
func NewBatchPrivateEndpointConnectionClient(client *armbatch.PrivateEndpointConnectionClient) BatchPrivateEndpointConnectionClient {
	return &batchPrivateEndpointConnectionClient{client: client}
}
