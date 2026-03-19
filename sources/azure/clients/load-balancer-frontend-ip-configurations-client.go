package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_load_balancer_frontend_ip_configurations_client.go -package=mocks -source=load-balancer-frontend-ip-configurations-client.go

// LoadBalancerFrontendIPConfigurationsPager is a type alias for the generic Pager interface.
type LoadBalancerFrontendIPConfigurationsPager = Pager[armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse]

// LoadBalancerFrontendIPConfigurationsClient is an interface for interacting with Azure load balancer frontend IP configurations.
type LoadBalancerFrontendIPConfigurationsClient interface {
	Get(ctx context.Context, resourceGroupName string, loadBalancerName string, frontendIPConfigurationName string) (armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse, error)
	NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerFrontendIPConfigurationsPager
}

type loadBalancerFrontendIPConfigurationsClient struct {
	client *armnetwork.LoadBalancerFrontendIPConfigurationsClient
}

func (a *loadBalancerFrontendIPConfigurationsClient) Get(ctx context.Context, resourceGroupName string, loadBalancerName string, frontendIPConfigurationName string) (armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, loadBalancerName, frontendIPConfigurationName, nil)
}

func (a *loadBalancerFrontendIPConfigurationsClient) NewListPager(resourceGroupName string, loadBalancerName string) LoadBalancerFrontendIPConfigurationsPager {
	return a.client.NewListPager(resourceGroupName, loadBalancerName, nil)
}

// NewLoadBalancerFrontendIPConfigurationsClient creates a new LoadBalancerFrontendIPConfigurationsClient from the Azure SDK client.
func NewLoadBalancerFrontendIPConfigurationsClient(client *armnetwork.LoadBalancerFrontendIPConfigurationsClient) LoadBalancerFrontendIPConfigurationsClient {
	return &loadBalancerFrontendIPConfigurationsClient{client: client}
}
