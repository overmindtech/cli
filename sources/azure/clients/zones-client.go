package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

//go:generate mockgen -destination=../shared/mocks/mock_zones_client.go -package=mocks -source=zones-client.go

// ZonesPager is a type alias for the generic Pager interface with zone response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type ZonesPager = Pager[armdns.ZonesClientListByResourceGroupResponse]

// ZonesClient is an interface for interacting with Azure zones
type ZonesClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armdns.ZonesClientListByResourceGroupOptions) ZonesPager
	Get(ctx context.Context, resourceGroupName string, zoneName string, options *armdns.ZonesClientGetOptions) (armdns.ZonesClientGetResponse, error)
}

type zonesClient struct {
	client *armdns.ZonesClient
}

func (a *zonesClient) NewListByResourceGroupPager(resourceGroupName string, options *armdns.ZonesClientListByResourceGroupOptions) ZonesPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *zonesClient) Get(ctx context.Context, resourceGroupName string, zoneName string, options *armdns.ZonesClientGetOptions) (armdns.ZonesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, zoneName, options)
}

// NewZonesClient creates a new ZonesClient from the Azure SDK client
func NewZonesClient(client *armdns.ZonesClient) ZonesClient {
	return &zonesClient{client: client}
}
