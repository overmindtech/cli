package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

//go:generate mockgen -destination=../shared/mocks/mock_public_ip_addresses_client.go -package=mocks -source=public-ip-addresses.go

// PublicIPAddressesPager is a type alias for the generic Pager interface with public IP address response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type PublicIPAddressesPager = Pager[armnetwork.PublicIPAddressesClientListResponse]

// PublicIPAddressesClient is an interface for interacting with Azure public IP addresses
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string) (armnetwork.PublicIPAddressesClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string) PublicIPAddressesPager
}

type publicIPAddressesClient struct {
	client *armnetwork.PublicIPAddressesClient
}

func (a *publicIPAddressesClient) Get(ctx context.Context, resourceGroupName string, publicIPAddressName string) (armnetwork.PublicIPAddressesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, publicIPAddressName, nil)
}

func (a *publicIPAddressesClient) List(ctx context.Context, resourceGroupName string) PublicIPAddressesPager {
	return a.client.NewListPager(resourceGroupName, nil)
}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient from the Azure SDK client
func NewPublicIPAddressesClient(client *armnetwork.PublicIPAddressesClient) PublicIPAddressesClient {
	return &publicIPAddressesClient{client: client}
}
