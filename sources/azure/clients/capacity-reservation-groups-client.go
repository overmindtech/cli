package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_capacity_reservation_groups_client.go -package=mocks -source=capacity-reservation-groups-client.go

// CapacityReservationGroupsPager is a type alias for the generic Pager interface with capacity reservation group response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type CapacityReservationGroupsPager = Pager[armcompute.CapacityReservationGroupsClientListByResourceGroupResponse]

// CapacityReservationGroupsClient is an interface for interacting with Azure capacity reservation groups
type CapacityReservationGroupsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.CapacityReservationGroupsClientListByResourceGroupOptions) CapacityReservationGroupsPager
	Get(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, options *armcompute.CapacityReservationGroupsClientGetOptions) (armcompute.CapacityReservationGroupsClientGetResponse, error)
}

type capacityReservationGroupsClient struct {
	client *armcompute.CapacityReservationGroupsClient
}

func (a *capacityReservationGroupsClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.CapacityReservationGroupsClientListByResourceGroupOptions) CapacityReservationGroupsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *capacityReservationGroupsClient) Get(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, options *armcompute.CapacityReservationGroupsClientGetOptions) (armcompute.CapacityReservationGroupsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, capacityReservationGroupName, options)
}

// NewCapacityReservationGroupsClient creates a new CapacityReservationGroupsClient from the Azure SDK client
func NewCapacityReservationGroupsClient(client *armcompute.CapacityReservationGroupsClient) CapacityReservationGroupsClient {
	return &capacityReservationGroupsClient{client: client}
}
