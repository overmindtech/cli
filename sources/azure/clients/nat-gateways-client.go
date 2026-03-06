package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_nat_gateways_client.go -package=mocks -source=nat-gateways-client.go

// NatGatewaysPager is a type alias for the generic Pager interface with NAT gateway list response type.
type NatGatewaysPager = Pager[armnetwork.NatGatewaysClientListResponse]

// NatGatewaysClient is an interface for interacting with Azure NAT gateways.
type NatGatewaysClient interface {
	Get(ctx context.Context, resourceGroupName string, natGatewayName string, options *armnetwork.NatGatewaysClientGetOptions) (armnetwork.NatGatewaysClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.NatGatewaysClientListOptions) NatGatewaysPager
}

type natGatewaysClient struct {
	client *armnetwork.NatGatewaysClient
}

func (c *natGatewaysClient) Get(ctx context.Context, resourceGroupName string, natGatewayName string, options *armnetwork.NatGatewaysClientGetOptions) (armnetwork.NatGatewaysClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, natGatewayName, options)
}

func (c *natGatewaysClient) NewListPager(resourceGroupName string, options *armnetwork.NatGatewaysClientListOptions) NatGatewaysPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewNatGatewaysClient creates a new NatGatewaysClient from the Azure SDK client.
func NewNatGatewaysClient(client *armnetwork.NatGatewaysClient) NatGatewaysClient {
	return &natGatewaysClient{client: client}
}
