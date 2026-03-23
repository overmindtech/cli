package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_load_balancer_backend_address_pools_client.go -package=mocks -source=load-balancer-backend-address-pools-client.go

// LoadBalancerBackendAddressPoolsPager is a type alias for the generic Pager interface.
type LoadBalancerBackendAddressPoolsPager = Pager[armnetwork.LoadBalancerBackendAddressPoolsClientListResponse]

// LoadBalancerBackendAddressPoolsClient is an interface for interacting with Azure load balancer backend address pools.
type LoadBalancerBackendAddressPoolsClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, backendAddressPoolName string) (armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse, error)
	NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerBackendAddressPoolsPager
}

type loadBalancerBackendAddressPoolsClient struct {
	client *armnetwork.LoadBalancerBackendAddressPoolsClient
}

func (a *loadBalancerBackendAddressPoolsClient) Get(ctx context.Context, resourceGroupName string, loadBalancerName string, backendAddressPoolName string) (armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, loadBalancerName, backendAddressPoolName, nil)
}

func (a *loadBalancerBackendAddressPoolsClient) NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerBackendAddressPoolsPager {
	return a.client.NewListPager(resourceGroupName, loadBalancerName, nil)
}

// NewLoadBalancerBackendAddressPoolsClient creates a new LoadBalancerBackendAddressPoolsClient from the Azure SDK client.
func NewLoadBalancerBackendAddressPoolsClient(client *armnetwork.LoadBalancerBackendAddressPoolsClient) LoadBalancerBackendAddressPoolsClient {
	return &loadBalancerBackendAddressPoolsClient{client: client}
}
