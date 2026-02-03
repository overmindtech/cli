package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
)

//go:generate mockgen -destination=../shared/mocks/mock_route_tables_client.go -package=mocks -source=route-tables-client.go

// RouteTablesPager is a type alias for the generic Pager interface with route table response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type RouteTablesPager = Pager[armnetwork.RouteTablesClientListResponse]

// RouteTablesClient is an interface for interacting with Azure route tables
type RouteTablesClient interface {
	Get(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientGetOptions) (armnetwork.RouteTablesClientGetResponse, error)
	List(resourceGroupName string, options *armnetwork.RouteTablesClientListOptions) RouteTablesPager
}

type routeTablesClient struct {
	client *armnetwork.RouteTablesClient
}

func (a *routeTablesClient) Get(ctx context.Context, resourceGroupName string, routeTableName string, options *armnetwork.RouteTablesClientGetOptions) (armnetwork.RouteTablesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, routeTableName, options)
}

func (a *routeTablesClient) List(resourceGroupName string, options *armnetwork.RouteTablesClientListOptions) RouteTablesPager {
	return a.client.NewListPager(resourceGroupName, options)
}

// NewRouteTablesClient creates a new RouteTablesClient from the Azure SDK client
func NewRouteTablesClient(client *armnetwork.RouteTablesClient) RouteTablesClient {
	return &routeTablesClient{client: client}
}
