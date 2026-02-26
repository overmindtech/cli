package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_routes_client.go -package=mocks -source=routes-client.go

// RoutesPager is a type alias for the generic Pager interface with routes list response type.
type RoutesPager = Pager[armnetwork.RoutesClientListResponse]

// RoutesClient is an interface for interacting with Azure routes (child of route table).
type RoutesClient interface {
	Get(ctx context.Context, resourceGroupName string, routeTableName string, routeName string, options *armnetwork.RoutesClientGetOptions) (armnetwork.RoutesClientGetResponse, error)
	NewListPager(resourceGroupName string, routeTableName string, options *armnetwork.RoutesClientListOptions) RoutesPager
}

type routesClient struct {
	client *armnetwork.RoutesClient
}

func (a *routesClient) Get(ctx context.Context, resourceGroupName string, routeTableName string, routeName string, options *armnetwork.RoutesClientGetOptions) (armnetwork.RoutesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, routeTableName, routeName, options)
}

func (a *routesClient) NewListPager(resourceGroupName string, routeTableName string, options *armnetwork.RoutesClientListOptions) RoutesPager {
	return a.client.NewListPager(resourceGroupName, routeTableName, options)
}

// NewRoutesClient creates a new RoutesClient from the Azure SDK client.
func NewRoutesClient(client *armnetwork.RoutesClient) RoutesClient {
	return &routesClient{client: client}
}
