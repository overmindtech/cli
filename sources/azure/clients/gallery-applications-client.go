package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_gallery_applications_client.go -package=mocks -source=gallery-applications-client.go

// GalleryApplicationsPager is a type alias for the generic Pager interface with gallery application response type.
type GalleryApplicationsPager = Pager[armcompute.GalleryApplicationsClientListByGalleryResponse]

// GalleryApplicationsClient is an interface for interacting with Azure gallery applications
type GalleryApplicationsClient interface {
	NewListByGalleryPager(resourceGroupName string, galleryName string, options *armcompute.GalleryApplicationsClientListByGalleryOptions) GalleryApplicationsPager
	Get(ctx context.Context, resourceGroupName string, galleryName string, galleryApplicationName string, options *armcompute.GalleryApplicationsClientGetOptions) (armcompute.GalleryApplicationsClientGetResponse, error)
}

type galleryApplicationsClient struct {
	client *armcompute.GalleryApplicationsClient
}

func (c *galleryApplicationsClient) NewListByGalleryPager(resourceGroupName string, galleryName string, options *armcompute.GalleryApplicationsClientListByGalleryOptions) GalleryApplicationsPager {
	return c.client.NewListByGalleryPager(resourceGroupName, galleryName, options)
}

func (c *galleryApplicationsClient) Get(ctx context.Context, resourceGroupName string, galleryName string, galleryApplicationName string, options *armcompute.GalleryApplicationsClientGetOptions) (armcompute.GalleryApplicationsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, galleryName, galleryApplicationName, options)
}

// NewGalleryApplicationsClient creates a new GalleryApplicationsClient from the Azure SDK client
func NewGalleryApplicationsClient(client *armcompute.GalleryApplicationsClient) GalleryApplicationsClient {
	return &galleryApplicationsClient{client: client}
}
