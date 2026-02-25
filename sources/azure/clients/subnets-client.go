package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_subnets_client.go -package=mocks -source=subnets-client.go

// SubnetsPager is a type alias for the generic Pager interface with subnet list response type.
type SubnetsPager = Pager[armnetwork.SubnetsClientListResponse]

// SubnetsClient is an interface for interacting with Azure virtual network subnets.
type SubnetsClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, options *armnetwork.SubnetsClientGetOptions) (armnetwork.SubnetsClientGetResponse, error)
	NewListPager(resourceGroupName string, virtualNetworkName string, options *armnetwork.SubnetsClientListOptions) SubnetsPager
}

type subnetsClientAdapter struct {
	client *armnetwork.SubnetsClient
}

func (a *subnetsClientAdapter) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, subnetName string, options *armnetwork.SubnetsClientGetOptions) (armnetwork.SubnetsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, virtualNetworkName, subnetName, options)
}

func (a *subnetsClientAdapter) NewListPager(resourceGroupName string, virtualNetworkName string, options *armnetwork.SubnetsClientListOptions) SubnetsPager {
	return a.client.NewListPager(resourceGroupName, virtualNetworkName, options)
}

// NewSubnetsClient creates a new SubnetsClient from the Azure SDK client.
func NewSubnetsClient(client *armnetwork.SubnetsClient) SubnetsClient {
	return &subnetsClientAdapter{client: client}
}
