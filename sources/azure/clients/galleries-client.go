package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_galleries_client.go -package=mocks -source=galleries-client.go

// GalleriesPager is a type alias for the generic Pager interface with gallery response type.
type GalleriesPager = Pager[armcompute.GalleriesClientListByResourceGroupResponse]

// GalleriesClient is an interface for interacting with Azure compute galleries
type GalleriesClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.GalleriesClientListByResourceGroupOptions) GalleriesPager
	Get(ctx context.Context, resourceGroupName string, galleryName string, options *armcompute.GalleriesClientGetOptions) (armcompute.GalleriesClientGetResponse, error)
}

type galleriesClient struct {
	client *armcompute.GalleriesClient
}

func (c *galleriesClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.GalleriesClientListByResourceGroupOptions) GalleriesPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (c *galleriesClient) Get(ctx context.Context, resourceGroupName string, galleryName string, options *armcompute.GalleriesClientGetOptions) (armcompute.GalleriesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, galleryName, options)
}

// NewGalleriesClient creates a new GalleriesClient from the Azure SDK client
func NewGalleriesClient(client *armcompute.GalleriesClient) GalleriesClient {
	return &galleriesClient{client: client}
}
