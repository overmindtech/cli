package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_proximity_placement_groups_client.go -package=mocks -source=proximity-placement-groups-client.go

// ProximityPlacementGroupsPager is a type alias for the generic Pager interface with proximity placement group response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type ProximityPlacementGroupsPager = Pager[armcompute.ProximityPlacementGroupsClientListByResourceGroupResponse]

// ProximityPlacementGroupsClient is an interface for interacting with Azure proximity placement groups
type ProximityPlacementGroupsClient interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armcompute.ProximityPlacementGroupsClientListByResourceGroupOptions) ProximityPlacementGroupsPager
	Get(ctx context.Context, resourceGroupName string, proximityPlacementGroupName string, options *armcompute.ProximityPlacementGroupsClientGetOptions) (armcompute.ProximityPlacementGroupsClientGetResponse, error)
}

type proximityPlacementGroupsClient struct {
	client *armcompute.ProximityPlacementGroupsClient
}

func (a *proximityPlacementGroupsClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armcompute.ProximityPlacementGroupsClientListByResourceGroupOptions) ProximityPlacementGroupsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *proximityPlacementGroupsClient) Get(ctx context.Context, resourceGroupName string, proximityPlacementGroupName string, options *armcompute.ProximityPlacementGroupsClientGetOptions) (armcompute.ProximityPlacementGroupsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, proximityPlacementGroupName, options)
}

// NewProximityPlacementGroupsClient creates a new ProximityPlacementGroupsClient from the Azure SDK client
func NewProximityPlacementGroupsClient(client *armcompute.ProximityPlacementGroupsClient) ProximityPlacementGroupsClient {
	return &proximityPlacementGroupsClient{client: client}
}
