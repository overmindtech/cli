package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_batch_pool_client.go -package=mocks -source=batch-pool-client.go

// BatchPoolsPager is a type alias for the generic Pager interface with batch pool response type.
type BatchPoolsPager = Pager[armbatch.PoolClientListByBatchAccountResponse]

// BatchPoolsClient is an interface for interacting with Azure Batch pools (child of Batch account).
type BatchPoolsClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, poolName string) (armbatch.PoolClientGetResponse, error)
	ListByBatchAccount(ctx context.Context, resourceGroupName string, accountName string) BatchPoolsPager
}

type batchPoolsClient struct {
	client *armbatch.PoolClient
}

func (c *batchPoolsClient) Get(ctx context.Context, resourceGroupName string, accountName string, poolName string) (armbatch.PoolClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, poolName, nil)
}

func (c *batchPoolsClient) ListByBatchAccount(ctx context.Context, resourceGroupName string, accountName string) BatchPoolsPager {
	return c.client.NewListByBatchAccountPager(resourceGroupName, accountName, nil)
}

// NewBatchPoolsClient creates a new BatchPoolsClient from the Azure SDK client.
func NewBatchPoolsClient(client *armbatch.PoolClient) BatchPoolsClient {
	return &batchPoolsClient{client: client}
}
