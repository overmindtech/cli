package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
)

//go:generate mockgen -destination=../shared/mocks/mock_elastic_san_volume_client.go -package=mocks -source=elastic-san-volume-client.go

// ElasticSanVolumePager is a type alias for the generic Pager interface with volume list response type.
type ElasticSanVolumePager = Pager[armelasticsan.VolumesClientListByVolumeGroupResponse]

// ElasticSanVolumeClient is an interface for interacting with Azure Elastic SAN volumes.
type ElasticSanVolumeClient interface {
	Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, volumeName string, options *armelasticsan.VolumesClientGetOptions) (armelasticsan.VolumesClientGetResponse, error)
	NewListByVolumeGroupPager(resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumesClientListByVolumeGroupOptions) ElasticSanVolumePager
}

type elasticSanVolumeClient struct {
	client *armelasticsan.VolumesClient
}

func (c *elasticSanVolumeClient) Get(ctx context.Context, resourceGroupName string, elasticSanName string, volumeGroupName string, volumeName string, options *armelasticsan.VolumesClientGetOptions) (armelasticsan.VolumesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, options)
}

func (c *elasticSanVolumeClient) NewListByVolumeGroupPager(resourceGroupName string, elasticSanName string, volumeGroupName string, options *armelasticsan.VolumesClientListByVolumeGroupOptions) ElasticSanVolumePager {
	return c.client.NewListByVolumeGroupPager(resourceGroupName, elasticSanName, volumeGroupName, options)
}

// NewElasticSanVolumeClient creates a new ElasticSanVolumeClient from the Azure SDK client.
func NewElasticSanVolumeClient(client *armelasticsan.VolumesClient) ElasticSanVolumeClient {
	return &elasticSanVolumeClient{client: client}
}
