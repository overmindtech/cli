package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_interface_ip_configurations_client.go -package=mocks -source=interface-ip-configurations-client.go

// InterfaceIPConfigurationsPager is a type alias for the generic Pager interface with InterfaceIPConfiguration response type.
type InterfaceIPConfigurationsPager = Pager[armnetwork.InterfaceIPConfigurationsClientListResponse]

// InterfaceIPConfigurationsClient is an interface for interacting with Azure network interface IP configurations
type InterfaceIPConfigurationsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkInterfaceName string, ipConfigurationName string) (armnetwork.InterfaceIPConfigurationsClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, networkInterfaceName string) InterfaceIPConfigurationsPager
}

type interfaceIPConfigurationsClient struct {
	client *armnetwork.InterfaceIPConfigurationsClient
}

func (a *interfaceIPConfigurationsClient) Get(ctx context.Context, resourceGroupName string, networkInterfaceName string, ipConfigurationName string) (armnetwork.InterfaceIPConfigurationsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, networkInterfaceName, ipConfigurationName, nil)
}

func (a *interfaceIPConfigurationsClient) List(ctx context.Context, resourceGroupName string, networkInterfaceName string) InterfaceIPConfigurationsPager {
	return a.client.NewListPager(resourceGroupName, networkInterfaceName, nil)
}

// NewInterfaceIPConfigurationsClient creates a new InterfaceIPConfigurationsClient from the Azure SDK client
func NewInterfaceIPConfigurationsClient(client *armnetwork.InterfaceIPConfigurationsClient) InterfaceIPConfigurationsClient {
	return &interfaceIPConfigurationsClient{client: client}
}
