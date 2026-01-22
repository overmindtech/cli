package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_images_client.go -package=mocks -source=images-client.go

// ImagesPager is a type alias for the generic Pager interface with image response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type ImagesPager = Pager[armcompute.ImagesClientListByResourceGroupResponse]

// ImagesClient is an interface for interacting with Azure images
type ImagesClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.ImagesClientListByResourceGroupOptions) ImagesPager
	Get(ctx context.Context, resourceGroupName string, imageName string, options *armcompute.ImagesClientGetOptions) (armcompute.ImagesClientGetResponse, error)
}

type imagesClient struct {
	client *armcompute.ImagesClient
}

func (a *imagesClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.ImagesClientListByResourceGroupOptions) ImagesPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *imagesClient) Get(ctx context.Context, resourceGroupName string, imageName string, options *armcompute.ImagesClientGetOptions) (armcompute.ImagesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, imageName, options)
}

// NewImagesClient creates a new ImagesClient from the Azure SDK client
func NewImagesClient(client *armcompute.ImagesClient) ImagesClient {
	return &imagesClient{client: client}
}
