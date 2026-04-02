package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_private_link_services_client.go -package=mocks -source=private-link-services-client.go

// PrivateLinkServicesPager is a type alias for the generic Pager interface with private link service response type.
type PrivateLinkServicesPager = Pager[armnetwork.PrivateLinkServicesClientListResponse]

// PrivateLinkServicesClient is an interface for interacting with Azure private link services.
type PrivateLinkServicesClient interface {
	Get(ctx context.Context, resourceGroupName string, serviceName string) (armnetwork.PrivateLinkServicesClientGetResponse, error)
	List(resourceGroupName string) PrivateLinkServicesPager
}

type privateLinkServicesClient struct {
	client *armnetwork.PrivateLinkServicesClient
}

func (c *privateLinkServicesClient) Get(ctx context.Context, resourceGroupName string, serviceName string) (armnetwork.PrivateLinkServicesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, serviceName, nil)
}

func (c *privateLinkServicesClient) List(resourceGroupName string) PrivateLinkServicesPager {
	return c.client.NewListPager(resourceGroupName, nil)
}

// NewPrivateLinkServicesClient creates a new PrivateLinkServicesClient from the Azure SDK client.
func NewPrivateLinkServicesClient(client *armnetwork.PrivateLinkServicesClient) PrivateLinkServicesClient {
	return &privateLinkServicesClient{client: client}
}
