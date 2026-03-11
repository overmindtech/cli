package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
)

//go:generate mockgen -destination=../shared/mocks/mock_elastic_san_volume_group_client.go -package=mocks -source=elastic-san-volume-group-client.go

// ElasticSanVolumeGroupPager is a type alias for the generic Pager interface with volume group list response type.
type ElasticSanVolumeGroupPager = Pager[armelasticsan.VolumeGroupsClientListByElasticSanResponse]

// ElasticSanVolumeGroupClient is an interface for interacting with Azure Elastic SAN volume groups.
type ElasticSanVolumeGroupClient interface {
	Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumeGroupsClientGetOptions) (armelasticsan.VolumeGroupsClientGetResponse, error)
	NewListByElasticSanPager(resourceGroupName string, elasticSanName string, options *armelasticsan.VolumeGroupsClientListByElasticSanOptions) ElasticSanVolumeGroupPager
}

type elasticSanVolumeGroupClient struct {
	client *armelasticsan.VolumeGroupsClient
}

func (c *elasticSanVolumeGroupClient) Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumeGroupsClientGetOptions) (armelasticsan.VolumeGroupsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, options)
}

func (c *elasticSanVolumeGroupClient) NewListByElasticSanPager(resourceGroupName string, elasticSanName string, options *armelasticsan.VolumeGroupsClientListByElasticSanOptions) ElasticSanVolumeGroupPager {
	return c.client.NewListByElasticSanPager(resourceGroupName, elasticSanName, options)
}

// NewElasticSanVolumeGroupClient creates a new ElasticSanVolumeGroupClient from the Azure SDK client.
func NewElasticSanVolumeGroupClient(client *armelasticsan.VolumeGroupsClient) ElasticSanVolumeGroupClient {
	return &elasticSanVolumeGroupClient{client: client}
}
