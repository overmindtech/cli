package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_storage_accounts_client.go -package=mocks -source=storage-accounts-client.go

// StorageAccountsPager is a type alias for the generic Pager interface with storage account response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type StorageAccountsPager = Pager[armstorage.AccountsClientListByResourceGroupResponse]

// StorageAccountsClient is an interface for interacting with Azure storage accounts
type StorageAccountsClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string) (armstorage.AccountsClientGetPropertiesResponse, error)
	List(resourceGroupName string) StorageAccountsPager
}

type storageAccountsClient struct {
	client *armstorage.AccountsClient
}

func (a *storageAccountsClient) Get(ctx context.Context, resourceGroupName string, accountName string) (armstorage.AccountsClientGetPropertiesResponse, error) {
	return a.client.GetProperties(ctx, resourceGroupName, accountName, nil)
}

func (a *storageAccountsClient) List(resourceGroupName string) StorageAccountsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, nil)
}

// NewStorageAccountsClient creates a new StorageAccountsClient from the Azure SDK client
func NewStorageAccountsClient(client *armstorage.AccountsClient) StorageAccountsClient {
	return &storageAccountsClient{client: client}
}
