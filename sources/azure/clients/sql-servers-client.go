package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_servers_client.go -package=mocks -source=sql-servers-client.go

// SqlServersPager is a type alias for the generic Pager interface with sql server response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type SqlServersPager = Pager[armsql.ServersClientListByResourceGroupResponse]

// SqlServersClient is an interface for interacting with Azure sql servers
type SqlServersClient interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armsql.ServersClientListByResourceGroupOptions) SqlServersPager
	Get(ctx context.Context, resourceGroupName string, serverName string, options *armsql.ServersClientGetOptions) (armsql.ServersClientGetResponse, error)
}

type sqlServersClient struct {
	client *armsql.ServersClient
}

func (a *sqlServersClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armsql.ServersClientListByResourceGroupOptions) SqlServersPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (a *sqlServersClient) Get(ctx context.Context, resourceGroupName string, serverName string, options *armsql.ServersClientGetOptions) (armsql.ServersClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, options)
}

// NewSqlServersClient creates a new SqlServersClient from the Azure SDK client
func NewSqlServersClient(client *armsql.ServersClient) SqlServersClient {
	return &sqlServersClient{client: client}
}
