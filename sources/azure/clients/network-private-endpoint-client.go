package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_network_private_endpoint_client.go -package=mocks -source=network-private-endpoint-client.go

// PrivateEndpointsPager is a type alias for the generic Pager interface with private endpoint response type.
type PrivateEndpointsPager = Pager[armnetwork.PrivateEndpointsClientListResponse]

// PrivateEndpointsClient is an interface for interacting with Azure private endpoints.
type PrivateEndpointsClient interface {
	Get(ctx context.Context, resourceGroupName string, privateEndpointName string) (armnetwork.PrivateEndpointsClientGetResponse, error)
	List(resourceGroupName string) PrivateEndpointsPager
}

type privateEndpointsClient struct {
	client *armnetwork.PrivateEndpointsClient
}

func (c *privateEndpointsClient) Get(ctx context.Context, resourceGroupName string, privateEndpointName string) (armnetwork.PrivateEndpointsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, privateEndpointName, nil)
}

func (c *privateEndpointsClient) List(resourceGroupName string) PrivateEndpointsPager {
	return c.client.NewListPager(resourceGroupName, nil)
}

// NewPrivateEndpointsClient creates a new PrivateEndpointsClient from the Azure SDK client.
func NewPrivateEndpointsClient(client *armnetwork.PrivateEndpointsClient) PrivateEndpointsClient {
	return &privateEndpointsClient{client: client}
}
