package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
)

//go:generate mockgen -destination=../shared/mocks/mock_private_dns_zones_client.go -package=mocks -source=private-dns-zones-client.go

// PrivateDNSZonesPager is a type alias for the generic Pager interface with private zone response type.
type PrivateDNSZonesPager = Pager[armprivatedns.PrivateZonesClientListByResourceGroupResponse]

// PrivateDNSZonesClient is an interface for interacting with Azure Private DNS zones.
type PrivateDNSZonesClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armprivatedns.PrivateZonesClientListByResourceGroupOptions) PrivateDNSZonesPager
	Get(ctx context.Context, resourceGroupName string, privateZoneName string, options *armprivatedns.PrivateZonesClientGetOptions) (armprivatedns.PrivateZonesClientGetResponse, error)
}

type privateDNSZonesClient struct {
	client *armprivatedns.PrivateZonesClient
}

func (c *privateDNSZonesClient) NewListByResourceGroupPager(resourceGroupName string, options *armprivatedns.PrivateZonesClientListByResourceGroupOptions) PrivateDNSZonesPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (c *privateDNSZonesClient) Get(ctx context.Context, resourceGroupName string, privateZoneName string, options *armprivatedns.PrivateZonesClientGetOptions) (armprivatedns.PrivateZonesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, privateZoneName, options)
}

// NewPrivateDNSZonesClient creates a new PrivateDNSZonesClient from the Azure SDK client.
func NewPrivateDNSZonesClient(client *armprivatedns.PrivateZonesClient) PrivateDNSZonesClient {
	return &privateDNSZonesClient{client: client}
}
