package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_batch_application_client.go -package=mocks -source=batch-application-client.go

// BatchApplicationsPager is a type alias for the generic Pager interface with batch application response type.
type BatchApplicationsPager = Pager[armbatch.ApplicationClientListResponse]

// BatchApplicationsClient is an interface for interacting with Azure Batch applications
type BatchApplicationsClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, applicationName string) (armbatch.ApplicationClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) BatchApplicationsPager
}

type batchApplicationsClient struct {
	client *armbatch.ApplicationClient
}

func (c *batchApplicationsClient) Get(ctx context.Context, resourceGroupName string, accountName string, applicationName string) (armbatch.ApplicationClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, applicationName, nil)
}

func (c *batchApplicationsClient) List(ctx context.Context, resourceGroupName string, accountName string) BatchApplicationsPager {
	return c.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewBatchApplicationsClient creates a new BatchApplicationsClient from the Azure SDK client
func NewBatchApplicationsClient(client *armbatch.ApplicationClient) BatchApplicationsClient {
	return &batchApplicationsClient{client: client}
}
