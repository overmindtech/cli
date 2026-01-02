package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_load_balancers_client.go -package=mocks -source=load-balancers-client.go

// LoadBalancersPager is a type alias for the generic Pager interface with load balancer response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type LoadBalancersPager = Pager[armnetwork.LoadBalancersClientListResponse]

// LoadBalancersClient is an interface for interacting with Azure load balancers
type LoadBalancersClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string) (armnetwork.LoadBalancersClientGetResponse, error)
	List(resourceGroupName string) LoadBalancersPager
}

type loadBalancersClient struct {
	client *armnetwork.LoadBalancersClient
}

func (a *loadBalancersClient) Get(ctx context.Context, resourceGroupName string, loadBalancerName string) (armnetwork.LoadBalancersClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, loadBalancerName, nil)
}

func (a *loadBalancersClient) List(resourceGroupName string) LoadBalancersPager {
	return a.client.NewListPager(resourceGroupName, nil)
}

// NewLoadBalancersClient creates a new LoadBalancersClient from the Azure SDK client
func NewLoadBalancersClient(client *armnetwork.LoadBalancersClient) LoadBalancersClient {
	return &loadBalancersClient{client: client}
}
