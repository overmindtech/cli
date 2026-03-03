package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_capacity_reservations_client.go -package=mocks -source=capacity-reservations-client.go

// CapacityReservationsPager is a type alias for the generic Pager interface with capacity reservations list response type.
type CapacityReservationsPager = Pager[armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse]

// CapacityReservationsClient is an interface for interacting with Azure capacity reservations
type CapacityReservationsClient interface {
	NewListByCapacityReservationGroupPager(resourceGroupName string, capacityReservationGroupName string, options *armcompute.CapacityReservationsClientListByCapacityReservationGroupOptions) CapacityReservationsPager
	Get(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, capacityReservationName string, options *armcompute.CapacityReservationsClientGetOptions) (armcompute.CapacityReservationsClientGetResponse, error)
}

type capacityReservationsClient struct {
	client *armcompute.CapacityReservationsClient
}

func (c *capacityReservationsClient) NewListByCapacityReservationGroupPager(resourceGroupName string, capacityReservationGroupName string, options *armcompute.CapacityReservationsClientListByCapacityReservationGroupOptions) CapacityReservationsPager {
	return c.client.NewListByCapacityReservationGroupPager(resourceGroupName, capacityReservationGroupName, options)
}

func (c *capacityReservationsClient) Get(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, capacityReservationName string, options *armcompute.CapacityReservationsClientGetOptions) (armcompute.CapacityReservationsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, capacityReservationGroupName, capacityReservationName, options)
}

// NewCapacityReservationsClient creates a new CapacityReservationsClient from the Azure SDK client
func NewCapacityReservationsClient(client *armcompute.CapacityReservationsClient) CapacityReservationsClient {
	return &capacityReservationsClient{client: client}
}
