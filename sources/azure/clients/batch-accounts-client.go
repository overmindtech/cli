package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch"
)

//go:generate mockgen -destination=../shared/mocks/mock_batch_accounts_client.go -package=mocks -source=batch-accounts-client.go

// BatchAccountsPager is a type alias for the generic Pager interface with batch account response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type BatchAccountsPager = Pager[armbatch.AccountClientListByResourceGroupResponse]

// BatchAccountsClient is an interface for interacting with Azure batch accounts
type BatchAccountsClient interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string) BatchAccountsPager
	Get(ctx context.Context, resourceGroupName string, accountName string) (armbatch.AccountClientGetResponse, error)
}

type batchAccountsClient struct {
	client *armbatch.AccountClient
}

func (c *batchAccountsClient) ListByResourceGroup(ctx context.Context, resourceGroupName string) BatchAccountsPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, nil)
}

func (c *batchAccountsClient) Get(ctx context.Context, resourceGroupName string, accountName string) (armbatch.AccountClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, nil)
}

// NewBatchAccountsClient creates a new BatchAccountsClient from the Azure SDK client
func NewBatchAccountsClient(client *armbatch.AccountClient) BatchAccountsClient {
	return &batchAccountsClient{client: client}
}
