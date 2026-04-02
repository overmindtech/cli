package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_local_network_gateways_client.go -package=mocks -source=local-network-gateways-client.go

// LocalNetworkGatewaysPager is a type alias for the generic Pager interface with local network gateway list response type.
type LocalNetworkGatewaysPager = Pager[armnetwork.LocalNetworkGatewaysClientListResponse]

// LocalNetworkGatewaysClient is an interface for interacting with Azure local network gateways.
type LocalNetworkGatewaysClient interface {
	Get(ctx context.Context, resourceGroupName string, localNetworkGatewayName string, options *armnetwork.LocalNetworkGatewaysClientGetOptions) (armnetwork.LocalNetworkGatewaysClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.LocalNetworkGatewaysClientListOptions) LocalNetworkGatewaysPager
}

type localNetworkGatewaysClient struct {
	client *armnetwork.LocalNetworkGatewaysClient
}

func (c *localNetworkGatewaysClient) Get(ctx context.Context, resourceGroupName string, localNetworkGatewayName string, options *armnetwork.LocalNetworkGatewaysClientGetOptions) (armnetwork.LocalNetworkGatewaysClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, localNetworkGatewayName, options)
}

func (c *localNetworkGatewaysClient) NewListPager(resourceGroupName string, options *armnetwork.LocalNetworkGatewaysClientListOptions) LocalNetworkGatewaysPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewLocalNetworkGatewaysClient creates a new LocalNetworkGatewaysClient from the Azure SDK client.
func NewLocalNetworkGatewaysClient(client *armnetwork.LocalNetworkGatewaysClient) LocalNetworkGatewaysClient {
	return &localNetworkGatewaysClient{client: client}
}
