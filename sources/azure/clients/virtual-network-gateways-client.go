package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_network_gateways_client.go -package=mocks -source=virtual-network-gateways-client.go

// VirtualNetworkGatewaysPager is a type alias for the generic Pager interface with virtual network gateway list response type.
type VirtualNetworkGatewaysPager = Pager[armnetwork.VirtualNetworkGatewaysClientListResponse]

// VirtualNetworkGatewaysClient is an interface for interacting with Azure virtual network gateways.
type VirtualNetworkGatewaysClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkGatewayName string, options *armnetwork.VirtualNetworkGatewaysClientGetOptions) (armnetwork.VirtualNetworkGatewaysClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworkGatewaysClientListOptions) VirtualNetworkGatewaysPager
}

type virtualNetworkGatewaysClient struct {
	client *armnetwork.VirtualNetworkGatewaysClient
}

func (c *virtualNetworkGatewaysClient) Get(ctx context.Context, resourceGroupName string, virtualNetworkGatewayName string, options *armnetwork.VirtualNetworkGatewaysClientGetOptions) (armnetwork.VirtualNetworkGatewaysClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, virtualNetworkGatewayName, options)
}

func (c *virtualNetworkGatewaysClient) NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworkGatewaysClientListOptions) VirtualNetworkGatewaysPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewVirtualNetworkGatewaysClient creates a new VirtualNetworkGatewaysClient from the Azure SDK client.
func NewVirtualNetworkGatewaysClient(client *armnetwork.VirtualNetworkGatewaysClient) VirtualNetworkGatewaysClient {
	return &virtualNetworkGatewaysClient{client: client}
}
