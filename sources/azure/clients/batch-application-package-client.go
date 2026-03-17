package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
)

//go:generate mockgen -destination=../shared/mocks/mock_batch_application_package_client.go -package=mocks -source=batch-application-package-client.go

// BatchApplicationPackagesPager is a type alias for the generic Pager interface with batch application package response type.
type BatchApplicationPackagesPager = Pager[armbatch.ApplicationPackageClientListResponse]

// BatchApplicationPackagesClient is an interface for interacting with Azure Batch application packages.
type BatchApplicationPackagesClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, applicationName string, versionName string) (armbatch.ApplicationPackageClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string, applicationName string) BatchApplicationPackagesPager
}

type batchApplicationPackagesClient struct {
	client *armbatch.ApplicationPackageClient
}

func (c *batchApplicationPackagesClient) Get(ctx context.Context, resourceGroupName string, accountName string, applicationName string, versionName string) (armbatch.ApplicationPackageClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, applicationName, versionName, nil)
}

func (c *batchApplicationPackagesClient) List(ctx context.Context, resourceGroupName string, accountName string, applicationName string) BatchApplicationPackagesPager {
	return c.client.NewListPager(resourceGroupName, accountName, applicationName, nil)
}

// NewBatchApplicationPackagesClient creates a new BatchApplicationPackagesClient from the Azure SDK client.
func NewBatchApplicationPackagesClient(client *armbatch.ApplicationPackageClient) BatchApplicationPackagesClient {
	return &batchApplicationPackagesClient{client: client}
}
