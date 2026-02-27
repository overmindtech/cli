package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_storage_private_endpoint_connection_client.go -package=mocks -source=storage-private-endpoint-connection-client.go

// PrivateEndpointConnectionsPager is a type alias for the generic Pager interface with storage private endpoint connection list response type.
type PrivateEndpointConnectionsPager = Pager[armstorage.PrivateEndpointConnectionsClientListResponse]

// StoragePrivateEndpointConnectionsClient is an interface for interacting with Azure storage account private endpoint connections.
type StoragePrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armstorage.PrivateEndpointConnectionsClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) PrivateEndpointConnectionsPager
}

type storagePrivateEndpointConnectionsClient struct {
	client *armstorage.PrivateEndpointConnectionsClient
}

func (c *storagePrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, accountName string, privateEndpointConnectionName string) (armstorage.PrivateEndpointConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, privateEndpointConnectionName, nil)
}

func (c *storagePrivateEndpointConnectionsClient) List(ctx context.Context, resourceGroupName string, accountName string) PrivateEndpointConnectionsPager {
	return c.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewStoragePrivateEndpointConnectionsClient creates a new StoragePrivateEndpointConnectionsClient from the Azure SDK client.
func NewStoragePrivateEndpointConnectionsClient(client *armstorage.PrivateEndpointConnectionsClient) StoragePrivateEndpointConnectionsClient {
	return &storagePrivateEndpointConnectionsClient{client: client}
}
