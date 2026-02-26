package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_network_peerings_client.go -package=mocks -source=virtual-network-peerings-client.go

// VirtualNetworkPeeringsPager is a type alias for the generic Pager interface with virtual network peerings list response type.
type VirtualNetworkPeeringsPager = Pager[armnetwork.VirtualNetworkPeeringsClientListResponse]

// VirtualNetworkPeeringsClient is an interface for interacting with Azure virtual network peerings.
type VirtualNetworkPeeringsClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, peeringName string, options *armnetwork.VirtualNetworkPeeringsClientGetOptions) (armnetwork.VirtualNetworkPeeringsClientGetResponse, error)
	NewListPager(resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworkPeeringsClientListOptions) VirtualNetworkPeeringsPager
}

type virtualNetworkPeeringsClientAdapter struct {
	client *armnetwork.VirtualNetworkPeeringsClient
}

func (a *virtualNetworkPeeringsClientAdapter) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, peeringName string, options *armnetwork.VirtualNetworkPeeringsClientGetOptions) (armnetwork.VirtualNetworkPeeringsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, virtualNetworkName, peeringName, options)
}

func (a *virtualNetworkPeeringsClientAdapter) NewListPager(resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworkPeeringsClientListOptions) VirtualNetworkPeeringsPager {
	return a.client.NewListPager(resourceGroupName, virtualNetworkName, options)
}

// NewVirtualNetworkPeeringsClient creates a new VirtualNetworkPeeringsClient from the Azure SDK client.
func NewVirtualNetworkPeeringsClient(client *armnetwork.VirtualNetworkPeeringsClient) VirtualNetworkPeeringsClient {
	return &virtualNetworkPeeringsClientAdapter{client: client}
}
