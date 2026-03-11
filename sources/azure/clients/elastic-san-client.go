package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
)

//go:generate mockgen -destination=../shared/mocks/mock_elastic_san_client.go -package=mocks -source=elastic-san-client.go

// ElasticSanPager is a type alias for the generic Pager interface with Elastic SAN list response type.
type ElasticSanPager = Pager[armelasticsan.ElasticSansClientListByResourceGroupResponse]

// ElasticSanClient is an interface for interacting with Azure Elastic SAN (pool) resources.
type ElasticSanClient interface {
	Get(ctx context.Context, resourceGroupName string, elasticSanName string, options *armelasticsan.ElasticSansClientGetOptions) (armelasticsan.ElasticSansClientGetResponse, error)
	NewListByResourceGroupPager(resourceGroupName string, options *armelasticsan.ElasticSansClientListByResourceGroupOptions) ElasticSanPager
}

type elasticSanClient struct {
	client *armelasticsan.ElasticSansClient
}

func (c *elasticSanClient) Get(ctx context.Context, resourceGroupName string, elasticSanName string, options *armelasticsan.ElasticSansClientGetOptions) (armelasticsan.ElasticSansClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, elasticSanName, options)
}

func (c *elasticSanClient) NewListByResourceGroupPager(resourceGroupName string, options *armelasticsan.ElasticSansClientListByResourceGroupOptions) ElasticSanPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewElasticSanClient creates a new ElasticSanClient from the Azure SDK client.
func NewElasticSanClient(client *armelasticsan.ElasticSansClient) ElasticSanClient {
	return &elasticSanClient{client: client}
}
