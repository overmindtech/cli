package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_load_balancer_probes_client.go -package=mocks -source=load-balancer-probes-client.go

type LoadBalancerProbesPager = Pager[armnetwork.LoadBalancerProbesClientListResponse]

type LoadBalancerProbesClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, probeName string) (armnetwork.LoadBalancerProbesClientGetResponse, error)
	NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerProbesPager
}

type loadBalancerProbesClient struct {
	client *armnetwork.LoadBalancerProbesClient
}

func (a *loadBalancerProbesClient) Get(ctx context.Context, resourceGroupName string, loadBalancerName string, probeName string) (armnetwork.LoadBalancerProbesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, loadBalancerName, probeName, nil)
}

func (a *loadBalancerProbesClient) NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerProbesPager {
	return a.client.NewListPager(resourceGroupName, loadBalancerName, nil)
}

func NewLoadBalancerProbesClient(client *armnetwork.LoadBalancerProbesClient) LoadBalancerProbesClient {
	return &loadBalancerProbesClient{client: client}
}
