package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_file_shares_client.go -package=mocks -source=fileshares-client.go

// FileSharesPager is a type alias for the generic Pager interface with file share response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type FileSharesPager = Pager[armstorage.FileSharesClientListResponse]

// FileSharesClient is an interface for interacting with Azure file shares
type FileSharesClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, shareName string) (armstorage.FileSharesClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) FileSharesPager
}

type fileSharesClient struct {
	client *armstorage.FileSharesClient
}

func (a *fileSharesClient) Get(ctx context.Context, resourceGroupName string, accountName string, shareName string) (armstorage.FileSharesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, accountName, shareName, nil)
}

func (a *fileSharesClient) List(ctx context.Context, resourceGroupName string, accountName string) FileSharesPager {
	return a.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewFileSharesClient creates a new FileSharesClient from the Azure SDK client
func NewFileSharesClient(client *armstorage.FileSharesClient) FileSharesClient {
	return &fileSharesClient{client: client}
}
