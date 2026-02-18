package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_gallery_images_client.go -package=mocks -source=gallery-images-client.go

// GalleryImagesPager is a type alias for the generic Pager interface with gallery image response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type GalleryImagesPager = Pager[armcompute.GalleryImagesClientListByGalleryResponse]

// GalleryImagesClient is an interface for interacting with Azure gallery image definitions
type GalleryImagesClient interface {
	NewListByGalleryPager(resourceGroupName string, galleryName string, options *armcompute.GalleryImagesClientListByGalleryOptions) GalleryImagesPager
	Get(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string, options *armcompute.GalleryImagesClientGetOptions) (armcompute.GalleryImagesClientGetResponse, error)
}

type galleryImagesClient struct {
	client *armcompute.GalleryImagesClient
}

func (c *galleryImagesClient) NewListByGalleryPager(resourceGroupName string, galleryName string, options *armcompute.GalleryImagesClientListByGalleryOptions) GalleryImagesPager {
	return c.client.NewListByGalleryPager(resourceGroupName, galleryName, options)
}

func (c *galleryImagesClient) Get(ctx context.Context, resourceGroupName string, galleryName string, galleryImageName string, options *armcompute.GalleryImagesClientGetOptions) (armcompute.GalleryImagesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, galleryName, galleryImageName, options)
}

// NewGalleryImagesClient creates a new GalleryImagesClient from the Azure SDK client
func NewGalleryImagesClient(client *armcompute.GalleryImagesClient) GalleryImagesClient {
	return &galleryImagesClient{client: client}
}
