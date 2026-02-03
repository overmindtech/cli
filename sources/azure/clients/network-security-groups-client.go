package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

//go:generate mockgen -destination=../shared/mocks/mock_network_security_groups_client.go -package=mocks -source=network-security-groups-client.go

// NetworkSecurityGroupsPager is a type alias for the generic Pager interface with network security group response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type NetworkSecurityGroupsPager = Pager[armnetwork.SecurityGroupsClientListResponse]

// NetworkSecurityGroupsClient is an interface for interacting with Azure network security groups
type NetworkSecurityGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientGetOptions) (armnetwork.SecurityGroupsClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, options *armnetwork.SecurityGroupsClientListOptions) NetworkSecurityGroupsPager
}

type networkSecurityGroupsClient struct {
	client *armnetwork.SecurityGroupsClient
}

func (a *networkSecurityGroupsClient) Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityGroupsClientGetOptions) (armnetwork.SecurityGroupsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, networkSecurityGroupName, options)
}

func (a *networkSecurityGroupsClient) List(ctx context.Context, resourceGroupName string, options *armnetwork.SecurityGroupsClientListOptions) NetworkSecurityGroupsPager {
	return a.client.NewListPager(resourceGroupName, options)
}

// NewNetworkSecurityGroupsClient creates a new NetworkSecurityGroupsClient from the Azure SDK client
func NewNetworkSecurityGroupsClient(client *armnetwork.SecurityGroupsClient) NetworkSecurityGroupsClient {
	return &networkSecurityGroupsClient{client: client}
}
