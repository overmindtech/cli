package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_application_gateways_client.go -package=mocks -source=application-gateways-client.go

// ApplicationGatewaysPager is a type alias for the generic Pager interface with application gateway response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type ApplicationGatewaysPager = Pager[armnetwork.ApplicationGatewaysClientListResponse]

// ApplicationGatewaysClient is an interface for interacting with Azure application gateways
type ApplicationGatewaysClient interface {
	Get(ctx context.Context, resourceGroupName string, applicationGatewayName string, options *armnetwork.ApplicationGatewaysClientGetOptions) (armnetwork.ApplicationGatewaysClientGetResponse, error)
	List(resourceGroupName string, options *armnetwork.ApplicationGatewaysClientListOptions) ApplicationGatewaysPager
}

type applicationGatewaysClient struct {
	client *armnetwork.ApplicationGatewaysClient
}

func (a *applicationGatewaysClient) Get(ctx context.Context, resourceGroupName string, applicationGatewayName string, options *armnetwork.ApplicationGatewaysClientGetOptions) (armnetwork.ApplicationGatewaysClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, applicationGatewayName, options)
}

func (a *applicationGatewaysClient) List(resourceGroupName string, options *armnetwork.ApplicationGatewaysClientListOptions) ApplicationGatewaysPager {
	return a.client.NewListPager(resourceGroupName, options)
}

// NewApplicationGatewaysClient creates a new ApplicationGatewaysClient from the Azure SDK client
func NewApplicationGatewaysClient(client *armnetwork.ApplicationGatewaysClient) ApplicationGatewaysClient {
	return &applicationGatewaysClient{client: client}
}
