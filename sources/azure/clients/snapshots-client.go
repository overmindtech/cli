package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_snapshots_client.go -package=mocks -source=snapshots-client.go

// SnapshotsPager is a type alias for the generic Pager interface with snapshot response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type SnapshotsPager = Pager[armcompute.SnapshotsClientListByResourceGroupResponse]

// SnapshotsClient is an interface for interacting with Azure snapshots
type SnapshotsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armcompute.SnapshotsClientListByResourceGroupOptions) SnapshotsPager
	Get(ctx context.Context, resourceGroupName string, snapshotName string, options *armcompute.SnapshotsClientGetOptions) (armcompute.SnapshotsClientGetResponse, error)
}

type snapshotsClient struct {
	client *armcompute.SnapshotsClient
}

func (a *snapshotsClient) NewListByResourceGroupPager(resourceGroupName string, options *armcompute.SnapshotsClientListByResourceGroupOptions) SnapshotsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *snapshotsClient) Get(ctx context.Context, resourceGroupName string, snapshotName string, options *armcompute.SnapshotsClientGetOptions) (armcompute.SnapshotsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, snapshotName, options)
}

// NewSnapshotsClient creates a new SnapshotsClient from the Azure SDK client
func NewSnapshotsClient(client *armcompute.SnapshotsClient) SnapshotsClient {
	return &snapshotsClient{client: client}
}
