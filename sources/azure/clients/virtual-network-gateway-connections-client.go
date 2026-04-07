package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_network_gateway_connections_client.go -package=mocks -source=virtual-network-gateway-connections-client.go

// VirtualNetworkGatewayConnectionsPager is a type alias for the generic Pager interface with virtual network gateway connection list response type.
type VirtualNetworkGatewayConnectionsPager = Pager[armnetwork.VirtualNetworkGatewayConnectionsClientListResponse]

// VirtualNetworkGatewayConnectionsClient is an interface for interacting with Azure virtual network gateway connections.
type VirtualNetworkGatewayConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkGatewayConnectionName string, options *armnetwork.VirtualNetworkGatewayConnectionsClientGetOptions) (armnetwork.VirtualNetworkGatewayConnectionsClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworkGatewayConnectionsClientListOptions) VirtualNetworkGatewayConnectionsPager
}

type virtualNetworkGatewayConnectionsClient struct {
	client *armnetwork.VirtualNetworkGatewayConnectionsClient
}

func (c *virtualNetworkGatewayConnectionsClient) Get(ctx context.Context, resourceGroupName string, virtualNetworkGatewayConnectionName string, options *armnetwork.VirtualNetworkGatewayConnectionsClientGetOptions) (armnetwork.VirtualNetworkGatewayConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, virtualNetworkGatewayConnectionName, options)
}

func (c *virtualNetworkGatewayConnectionsClient) NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworkGatewayConnectionsClientListOptions) VirtualNetworkGatewayConnectionsPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewVirtualNetworkGatewayConnectionsClient creates a new VirtualNetworkGatewayConnectionsClient from the Azure SDK client.
func NewVirtualNetworkGatewayConnectionsClient(client *armnetwork.VirtualNetworkGatewayConnectionsClient) VirtualNetworkGatewayConnectionsClient {
	return &virtualNetworkGatewayConnectionsClient{client: client}
}
