package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_dedicated_host_groups_client.go -package=mocks -source=dedicated-host-groups-client.go

// DedicatedHostGroupsPager is a type alias for the generic Pager interface with dedicated host group response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type DedicatedHostGroupsPager = Pager[armcompute.DedicatedHostGroupsClientListByResourceGroupResponse]

// DedicatedHostGroupsClient is an interface for interacting with Azure dedicated host groups
type DedicatedHostGroupsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DedicatedHostGroupsClientListByResourceGroupOptions) DedicatedHostGroupsPager
	Get(ctx context.Context, resourceGroupName string, dedicatedHostGroupName string, options *armcompute.DedicatedHostGroupsClientGetOptions) (armcompute.DedicatedHostGroupsClientGetResponse, error)
}

type dedicatedHostGroupsClient struct {
	client *armcompute.DedicatedHostGroupsClient
}

func (a *dedicatedHostGroupsClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.DedicatedHostGroupsClientListByResourceGroupOptions) DedicatedHostGroupsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *dedicatedHostGroupsClient) Get(ctx context.Context, resourceGroupName string, dedicatedHostGroupName string, options *armcompute.DedicatedHostGroupsClientGetOptions) (armcompute.DedicatedHostGroupsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, dedicatedHostGroupName, options)
}

// NewDedicatedHostGroupsClient creates a new DedicatedHostGroupsClient from the Azure SDK client
func NewDedicatedHostGroupsClient(client *armcompute.DedicatedHostGroupsClient) DedicatedHostGroupsClient {
	return &dedicatedHostGroupsClient{client: client}
}
