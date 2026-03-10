package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
)

//go:generate mockgen -destination=../shared/mocks/mock_elastic_san_volume_snapshot_client.go -package=mocks -source=elastic-san-volume-snapshot-client.go

// ElasticSanVolumeSnapshotPager is a type alias for the generic Pager interface with volume snapshot list response type.
type ElasticSanVolumeSnapshotPager = Pager[armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse]

// ElasticSanVolumeSnapshotClient is an interface for interacting with Azure Elastic SAN volume snapshots.
type ElasticSanVolumeSnapshotClient interface {
	Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, snapshotName string, options *armelasticsan.VolumeSnapshotsClientGetOptions) (armelasticsan.VolumeSnapshotsClientGetResponse, error)
	ListByVolumeGroup(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumeSnapshotsClientListByVolumeGroupOptions) ElasticSanVolumeSnapshotPager
}

type elasticSanVolumeSnapshotClient struct {
	client *armelasticsan.VolumeSnapshotsClient
}

func (c *elasticSanVolumeSnapshotClient) Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, snapshotName string, options *armelasticsan.VolumeSnapshotsClientGetOptions) (armelasticsan.VolumeSnapshotsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, snapshotName, options)
}

func (c *elasticSanVolumeSnapshotClient) ListByVolumeGroup(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumeSnapshotsClientListByVolumeGroupOptions) ElasticSanVolumeSnapshotPager {
	return c.client.NewListByVolumeGroupPager(resourceGroupName, elasticSanName, volumeGroupName, options)
}

// NewElasticSanVolumeSnapshotClient creates a new ElasticSanVolumeSnapshotClient from the Azure SDK client.
func NewElasticSanVolumeSnapshotClient(client *armelasticsan.VolumeSnapshotsClient) ElasticSanVolumeSnapshotClient {
	return &elasticSanVolumeSnapshotClient{client: client}
}
