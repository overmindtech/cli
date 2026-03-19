package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_disk_accesses_client.go -package=mocks -source=disk-accesses-client.go

// DiskAccessesPager is a type alias for the generic Pager interface with disk access response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type DiskAccessesPager = Pager[armcompute.DiskAccessesClientListByResourceGroupResponse]

// DiskAccessesClient is an interface for interacting with Azure disk access
type DiskAccessesClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DiskAccessesClientListByResourceGroupOptions) DiskAccessesPager
	Get(ctx context.Context, resourceGroupName string, diskAccessName string, options *armcompute.DiskAccessesClientGetOptions) (armcompute.DiskAccessesClientGetResponse, error)
}

type diskAccessesClient struct {
	client *armcompute.DiskAccessesClient
}

func (a *diskAccessesClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DiskAccessesClientListByResourceGroupOptions) DiskAccessesPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *diskAccessesClient) Get(ctx context.Context, resourceGroupName string, diskAccessName string, options *armcompute.DiskAccessesClientGetOptions) (armcompute.DiskAccessesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, diskAccessName, options)
}

// NewDiskAccessesClient creates a new DiskAccessesClient from the Azure SDK client
func NewDiskAccessesClient(client *armcompute.DiskAccessesClient) DiskAccessesClient {
	return &diskAccessesClient{client: client}
}
