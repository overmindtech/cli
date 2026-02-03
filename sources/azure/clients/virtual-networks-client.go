package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_networks_client.go -package=mocks -source=virtual-networks-client.go

// VirtualNetworksPager is a type alias for the generic Pager interface with virtual network response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type VirtualNetworksPager = Pager[armnetwork.VirtualNetworksClientListResponse]

// VirtualNetworksClient is an interface for interacting with Azure virtual networks
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (armnetwork.VirtualNetworksClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworksClientListOptions) VirtualNetworksPager
}

// virtualNetworksClientAdapter adapts the concrete Azure SDK client to our interface
type virtualNetworksClientAdapter struct {
	client *armnetwork.VirtualNetworksClient
}

func (a *virtualNetworksClientAdapter) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, options *armnetwork.VirtualNetworksClientGetOptions) (armnetwork.VirtualNetworksClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, virtualNetworkName, options)
}

func (a *virtualNetworksClientAdapter) NewListPager(resourceGroupName string, options *armnetwork.VirtualNetworksClientListOptions) VirtualNetworksPager {
	return a.client.NewListPager(resourceGroupName, options)
}

// NewVirtualNetworksClient creates a new VirtualNetworksClient from the Azure SDK client
func NewVirtualNetworksClient(client *armnetwork.VirtualNetworksClient) VirtualNetworksClient {
	return &virtualNetworksClientAdapter{client: client}
}
