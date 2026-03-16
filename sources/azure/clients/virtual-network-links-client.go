package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_network_links_client.go -package=mocks -source=virtual-network-links-client.go

type VirtualNetworkLinksPager = Pager[armprivatedns.VirtualNetworkLinksClientListResponse]

type VirtualNetworkLinksClient interface {
	NewListPager(resourceGroupName string, privateZoneName string, options *armprivatedns.VirtualNetworkLinksClientListOptions) VirtualNetworkLinksPager
	Get(ctx context.Context, resourceGroupName string, privateZoneName string, virtualNetworkLinkName string, options *armprivatedns.VirtualNetworkLinksClientGetOptions) (armprivatedns.VirtualNetworkLinksClientGetResponse, error)
}

type virtualNetworkLinksClient struct {
	client *armprivatedns.VirtualNetworkLinksClient
}

func (c *virtualNetworkLinksClient) NewListPager(resourceGroupName string, privateZoneName string, options *armprivatedns.VirtualNetworkLinksClientListOptions) VirtualNetworkLinksPager {
	return c.client.NewListPager(resourceGroupName, privateZoneName, options)
}

func (c *virtualNetworkLinksClient) Get(ctx context.Context, resourceGroupName string, privateZoneName string, virtualNetworkLinkName string, options *armprivatedns.VirtualNetworkLinksClientGetOptions) (armprivatedns.VirtualNetworkLinksClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, privateZoneName, virtualNetworkLinkName, options)
}

func NewVirtualNetworkLinksClient(client *armprivatedns.VirtualNetworkLinksClient) VirtualNetworkLinksClient {
	return &virtualNetworkLinksClient{client: client}
}
