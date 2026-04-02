package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_ip_groups_client.go -package=mocks -source=ip-groups-client.go

// IPGroupsPager is a type alias for the generic Pager interface with IP groups response type.
type IPGroupsPager = Pager[armnetwork.IPGroupsClientListByResourceGroupResponse]

// IPGroupsClient is an interface for interacting with Azure IP Groups.
type IPGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, ipGroupsName string, options *armnetwork.IPGroupsClientGetOptions) (armnetwork.IPGroupsClientGetResponse, error)
	NewListByResourceGroupPager(resourceGroupName string, options *armnetwork.IPGroupsClientListByResourceGroupOptions) IPGroupsPager
}

type ipGroupsClient struct {
	client *armnetwork.IPGroupsClient
}

func (c *ipGroupsClient) Get(ctx context.Context, resourceGroupName string, ipGroupsName string, options *armnetwork.IPGroupsClientGetOptions) (armnetwork.IPGroupsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, ipGroupsName, options)
}

func (c *ipGroupsClient) NewListByResourceGroupPager(resourceGroupName string, options *armnetwork.IPGroupsClientListByResourceGroupOptions) IPGroupsPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewIPGroupsClient creates a new IPGroupsClient from the Azure SDK client.
func NewIPGroupsClient(client *armnetwork.IPGroupsClient) IPGroupsClient {
	return &ipGroupsClient{client: client}
}
