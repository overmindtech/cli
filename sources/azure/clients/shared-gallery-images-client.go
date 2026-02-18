package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_shared_gallery_images_client.go -package=mocks -source=shared-gallery-images-client.go

type SharedGalleryImagesPager = Pager[armcompute.SharedGalleryImagesClientListResponse]

type SharedGalleryImagesClient interface {
	NewListPager(location string, galleryUniqueName string, options *armcompute.SharedGalleryImagesClientListOptions) SharedGalleryImagesPager
	Get(ctx context.Context, location string, galleryUniqueName string, galleryImageName string, options *armcompute.SharedGalleryImagesClientGetOptions) (armcompute.SharedGalleryImagesClientGetResponse, error)
}

type sharedGalleryImagesClient struct {
	client *armcompute.SharedGalleryImagesClient
}

func (c *sharedGalleryImagesClient) NewListPager(location string, galleryUniqueName string, options *armcompute.SharedGalleryImagesClientListOptions) SharedGalleryImagesPager {
	return c.client.NewListPager(location, galleryUniqueName, options)
}

func (c *sharedGalleryImagesClient) Get(ctx context.Context, location string, galleryUniqueName string, galleryImageName string, options *armcompute.SharedGalleryImagesClientGetOptions) (armcompute.SharedGalleryImagesClientGetResponse, error) {
	return c.client.Get(ctx, location, galleryUniqueName, galleryImageName, options)
}

func NewSharedGalleryImagesClient(client *armcompute.SharedGalleryImagesClient) SharedGalleryImagesClient {
	return &sharedGalleryImagesClient{client: client}
}
