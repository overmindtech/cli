package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_disk_encryption_sets_client.go -package=mocks -source=disk-encryption-sets-client.go

// DiskEncryptionSetsPager is a type alias for the generic Pager interface with disk encryption set response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type DiskEncryptionSetsPager = Pager[armcompute.DiskEncryptionSetsClientListByResourceGroupResponse]

// DiskEncryptionSetsClient is an interface for interacting with Azure disk encryption sets
type DiskEncryptionSetsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DiskEncryptionSetsClientListByResourceGroupOptions) DiskEncryptionSetsPager
	Get(ctx context.Context, resourceGroupName string, diskEncryptionSetName string, options *armcompute.DiskEncryptionSetsClientGetOptions) (armcompute.DiskEncryptionSetsClientGetResponse, error)
}

type diskEncryptionSetsClient struct {
	client *armcompute.DiskEncryptionSetsClient
}

func (a *diskEncryptionSetsClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DiskEncryptionSetsClientListByResourceGroupOptions) DiskEncryptionSetsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *diskEncryptionSetsClient) Get(ctx context.Context, resourceGroupName string, diskEncryptionSetName string, options *armcompute.DiskEncryptionSetsClientGetOptions) (armcompute.DiskEncryptionSetsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, diskEncryptionSetName, options)
}

// NewDiskEncryptionSetsClient creates a new DiskEncryptionSetsClient from the Azure SDK client
func NewDiskEncryptionSetsClient(client *armcompute.DiskEncryptionSetsClient) DiskEncryptionSetsClient {
	return &diskEncryptionSetsClient{client: client}
}
