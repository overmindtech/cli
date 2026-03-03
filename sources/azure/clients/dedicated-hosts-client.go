package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_dedicated_hosts_client.go -package=mocks -source=dedicated-hosts-client.go

// DedicatedHostsPager is a type alias for the generic Pager interface with dedicated hosts list response type.
type DedicatedHostsPager = Pager[armcompute.DedicatedHostsClientListByHostGroupResponse]

// DedicatedHostsClient is an interface for interacting with Azure dedicated hosts
type DedicatedHostsClient interface {
	NewListByHostGroupPager(resourceGroupName string, hostGroupName string, options *armcompute.DedicatedHostsClientListByHostGroupOptions) DedicatedHostsPager
	Get(ctx context.Context, resourceGroupName string, hostGroupName string, hostName string, options *armcompute.DedicatedHostsClientGetOptions) (armcompute.DedicatedHostsClientGetResponse, error)
}

type dedicatedHostsClient struct {
	client *armcompute.DedicatedHostsClient
}

func (c *dedicatedHostsClient) NewListByHostGroupPager(resourceGroupName string, hostGroupName string, options *armcompute.DedicatedHostsClientListByHostGroupOptions) DedicatedHostsPager {
	return c.client.NewListByHostGroupPager(resourceGroupName, hostGroupName, options)
}

func (c *dedicatedHostsClient) Get(ctx context.Context, resourceGroupName string, hostGroupName string, hostName string, options *armcompute.DedicatedHostsClientGetOptions) (armcompute.DedicatedHostsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, hostGroupName, hostName, options)
}

// NewDedicatedHostsClient creates a new DedicatedHostsClient from the Azure SDK client
func NewDedicatedHostsClient(client *armcompute.DedicatedHostsClient) DedicatedHostsClient {
	return &dedicatedHostsClient{client: client}
}
