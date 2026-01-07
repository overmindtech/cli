package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_disks_client.go -package=mocks -source=disks-client.go

// DisksPager is a type alias for the generic Pager interface with disk response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type DisksPager = Pager[armcompute.DisksClientListByResourceGroupResponse]

// DisksClient is an interface for interacting with Azure disks
type DisksClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DisksClientListByResourceGroupOptions) DisksPager
	Get(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error)
}

type disksClient struct {
	client *armcompute.DisksClient
}

func (a *disksClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DisksClientListByResourceGroupOptions) DisksPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *disksClient) Get(ctx context.Context, resourceGroupName string, diskName string, options *armcompute.DisksClientGetOptions) (armcompute.DisksClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, diskName, options)
}

// NewDisksClient creates a new DisksClient from the Azure SDK client
func NewDisksClient(client *armcompute.DisksClient) DisksClient {
	return &disksClient{client: client}
}
