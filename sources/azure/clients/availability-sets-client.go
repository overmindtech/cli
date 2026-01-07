package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_availability_sets_client.go -package=mocks -source=availability-sets-client.go

// AvailabilitySetsPager is a type alias for the generic Pager interface with availability set response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type AvailabilitySetsPager = Pager[armcompute.AvailabilitySetsClientListResponse]

// AvailabilitySetsClient is an interface for interacting with Azure availability sets
type AvailabilitySetsClient interface {
	NewListPager(resourceGroupName string, options *armcompute.AvailabilitySetsClientListOptions) AvailabilitySetsPager
	Get(ctx context.Context, resourceGroupName string, availabilitySetName string, options *armcompute.AvailabilitySetsClientGetOptions) (armcompute.AvailabilitySetsClientGetResponse, error)
}

type availabilitySetsClient struct {
	client *armcompute.AvailabilitySetsClient
}

func (a *availabilitySetsClient) NewListPager(resourceGroupName string, options *armcompute.AvailabilitySetsClientListOptions) AvailabilitySetsPager {
	return a.client.NewListPager(resourceGroupName, options)
}

func (a *availabilitySetsClient) Get(ctx context.Context, resourceGroupName string, availabilitySetName string, options *armcompute.AvailabilitySetsClientGetOptions) (armcompute.AvailabilitySetsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, availabilitySetName, options)
}

// NewAvailabilitySetsClient creates a new AvailabilitySetsClient from the Azure SDK client
func NewAvailabilitySetsClient(client *armcompute.AvailabilitySetsClient) AvailabilitySetsClient {
	return &availabilitySetsClient{client: client}
}
