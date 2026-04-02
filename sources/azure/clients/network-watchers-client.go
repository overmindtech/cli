package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_network_watchers_client.go -package=mocks -source=network-watchers-client.go

// NetworkWatchersPager is a type alias for the generic Pager interface with network watchers response type.
type NetworkWatchersPager = Pager[armnetwork.WatchersClientListResponse]

// NetworkWatchersClient is an interface for interacting with Azure Network Watchers
type NetworkWatchersClient interface {
	NewListPager(resourceGroupName string, options *armnetwork.WatchersClientListOptions) NetworkWatchersPager
	Get(ctx context.Context, resourceGroupName string, networkWatcherName string, options *armnetwork.WatchersClientGetOptions) (armnetwork.WatchersClientGetResponse, error)
}

type networkWatchersClient struct {
	client *armnetwork.WatchersClient
}

func (c *networkWatchersClient) NewListPager(resourceGroupName string, options *armnetwork.WatchersClientListOptions) NetworkWatchersPager {
	return c.client.NewListPager(resourceGroupName, options)
}

func (c *networkWatchersClient) Get(ctx context.Context, resourceGroupName string, networkWatcherName string, options *armnetwork.WatchersClientGetOptions) (armnetwork.WatchersClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, networkWatcherName, options)
}

// NewNetworkWatchersClient creates a new NetworkWatchersClient from the Azure SDK client
func NewNetworkWatchersClient(client *armnetwork.WatchersClient) NetworkWatchersClient {
	return &networkWatchersClient{client: client}
}
