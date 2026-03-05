package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_public_ip_prefixes_client.go -package=mocks -source=public-ip-prefixes-client.go

// PublicIPPrefixesPager is a type alias for the generic Pager interface with public IP prefix response type.
type PublicIPPrefixesPager = Pager[armnetwork.PublicIPPrefixesClientListResponse]

// PublicIPPrefixesClient is an interface for interacting with Azure public IP prefixes.
type PublicIPPrefixesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPPrefixName string, options *armnetwork.PublicIPPrefixesClientGetOptions) (armnetwork.PublicIPPrefixesClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.PublicIPPrefixesClientListOptions) PublicIPPrefixesPager
}

type publicIPPrefixesClient struct {
	client *armnetwork.PublicIPPrefixesClient
}

func (c *publicIPPrefixesClient) Get(ctx context.Context, resourceGroupName string, publicIPPrefixName string, options *armnetwork.PublicIPPrefixesClientGetOptions) (armnetwork.PublicIPPrefixesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, publicIPPrefixName, options)
}

func (c *publicIPPrefixesClient) NewListPager(resourceGroupName string, options *armnetwork.PublicIPPrefixesClientListOptions) PublicIPPrefixesPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewPublicIPPrefixesClient creates a new PublicIPPrefixesClient from the Azure SDK client.
func NewPublicIPPrefixesClient(client *armnetwork.PublicIPPrefixesClient) PublicIPPrefixesClient {
	return &publicIPPrefixesClient{client: client}
}
