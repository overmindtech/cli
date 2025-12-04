package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

//go:generate mockgen -destination=../shared/mocks/mock_blob_containers_client.go -package=mocks -source=blob-containers-client.go

// BlobContainersPager is an interface for paging through blob container results
type BlobContainersPager interface {
	More() bool
	NextPage(ctx context.Context) (armstorage.BlobContainersClientListResponse, error)
}

// BlobContainersClient is an interface for interacting with Azure blob containers
type BlobContainersClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, containerName string) (armstorage.BlobContainersClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) BlobContainersPager
}

type blobContainersClient struct {
	client *armstorage.BlobContainersClient
}

func (a *blobContainersClient) Get(ctx context.Context, resourceGroupName string, accountName string, containerName string) (armstorage.BlobContainersClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, accountName, containerName, nil)
}

func (a *blobContainersClient) List(ctx context.Context, resourceGroupName string, accountName string) BlobContainersPager {
	return a.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewBlobContainersClient creates a new BlobContainersClient from the Azure SDK client
func NewBlobContainersClient(client *armstorage.BlobContainersClient) BlobContainersClient {
	return &blobContainersClient{client: client}
}
