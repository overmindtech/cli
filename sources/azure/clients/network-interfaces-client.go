package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_network_interfaces_client.go -package=mocks -source=network-interfaces-client.go

// NetworkInterfacesPager is a type alias for the generic Pager interface with network interface response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type NetworkInterfacesPager = Pager[armnetwork.InterfacesClientListResponse]

// NetworkInterfacesClient is an interface for interacting with Azure network interfaces
type NetworkInterfacesClient interface {
	Get(ctx context.Context, resourceGroupName string, networkInterfaceName string) (armnetwork.InterfacesClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string) NetworkInterfacesPager
}

type networkInterfacesClient struct {
	client *armnetwork.InterfacesClient
}

func (a *networkInterfacesClient) Get(ctx context.Context, resourceGroupName string, networkInterfaceName string) (armnetwork.InterfacesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, networkInterfaceName, nil)
}

func (a *networkInterfacesClient) List(ctx context.Context, resourceGroupName string) NetworkInterfacesPager {
	return a.client.NewListPager(resourceGroupName, nil)
}

// NewNetworkInterfacesClient creates a new NetworkInterfacesClient from the Azure SDK client
func NewNetworkInterfacesClient(client *armnetwork.InterfacesClient) NetworkInterfacesClient {
	return &networkInterfacesClient{client: client}
}
