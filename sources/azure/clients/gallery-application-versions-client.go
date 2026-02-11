package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_gallery_application_versions_client.go -package=mocks -source=gallery-application-versions-client.go

// GalleryApplicationVersionsPager is a type alias for the generic Pager interface with gallery application version response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type GalleryApplicationVersionsPager = Pager[armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse]

// GalleryApplicationVersionsClient is an interface for interacting with Azure gallery application versions
type GalleryApplicationVersionsClient interface {
	NewListByGalleryApplicationPager(resourceGroupName string, galleryName string, galleryApplicationName string, options *armcompute.GalleryApplicationVersionsClientListByGalleryApplicationOptions) GalleryApplicationVersionsPager
	Get(ctx context.Context, resourceGroupName string, galleryName string, galleryApplicationName string, galleryApplicationVersionName string, options *armcompute.GalleryApplicationVersionsClientGetOptions) (armcompute.GalleryApplicationVersionsClientGetResponse, error)
}

type galleryApplicationVersionsClient struct {
	client *armcompute.GalleryApplicationVersionsClient
}

func (c *galleryApplicationVersionsClient) NewListByGalleryApplicationPager(resourceGroupName string, galleryName string, galleryApplicationName string, options *armcompute.GalleryApplicationVersionsClientListByGalleryApplicationOptions) GalleryApplicationVersionsPager {
	return c.client.NewListByGalleryApplicationPager(resourceGroupName, galleryName, galleryApplicationName, options)
}

func (c *galleryApplicationVersionsClient) Get(ctx context.Context, resourceGroupName string, galleryName string, galleryApplicationName string, galleryApplicationVersionName string, options *armcompute.GalleryApplicationVersionsClientGetOptions) (armcompute.GalleryApplicationVersionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, galleryName, galleryApplicationName, galleryApplicationVersionName, options)
}

// NewGalleryApplicationVersionsClient creates a new GalleryApplicationVersionsClient from the Azure SDK client
func NewGalleryApplicationVersionsClient(client *armcompute.GalleryApplicationVersionsClient) GalleryApplicationVersionsClient {
	return &galleryApplicationVersionsClient{client: client}
}
