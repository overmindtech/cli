package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_elastic_pool_client.go -package=mocks -source=sql-elastic-pool-client.go

// SqlElasticPoolPager is a type alias for the generic Pager interface with SQL elastic pool list response type.
type SqlElasticPoolPager = Pager[armsql.ElasticPoolsClientListByServerResponse]

// SqlElasticPoolClient is an interface for interacting with Azure SQL elastic pools.
type SqlElasticPoolClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlElasticPoolPager
	Get(ctx context.Context, resourceGroupName string, serverName string, elasticPoolName string) (armsql.ElasticPoolsClientGetResponse, error)
}

type sqlElasticPoolClient struct {
	client *armsql.ElasticPoolsClient
}

func (a *sqlElasticPoolClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlElasticPoolPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlElasticPoolClient) Get(ctx context.Context, resourceGroupName string, serverName string, elasticPoolName string) (armsql.ElasticPoolsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, elasticPoolName, nil)
}

// NewSqlElasticPoolClient creates a new SqlElasticPoolClient from the Azure SDK client.
func NewSqlElasticPoolClient(client *armsql.ElasticPoolsClient) SqlElasticPoolClient {
	return &sqlElasticPoolClient{client: client}
}
