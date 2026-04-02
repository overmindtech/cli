package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_failover_groups_client.go -package=mocks -source=sql-failover-groups-client.go

// SqlFailoverGroupsPager is a type alias for the generic Pager interface with failover groups response type.
type SqlFailoverGroupsPager = Pager[armsql.FailoverGroupsClientListByServerResponse]

// SqlFailoverGroupsClient is an interface for interacting with Azure SQL Server Failover Groups
type SqlFailoverGroupsClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlFailoverGroupsPager
	Get(ctx context.Context, resourceGroupName string, serverName string, failoverGroupName string) (armsql.FailoverGroupsClientGetResponse, error)
}

type sqlFailoverGroupsClient struct {
	client *armsql.FailoverGroupsClient
}

func (a *sqlFailoverGroupsClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlFailoverGroupsPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlFailoverGroupsClient) Get(ctx context.Context, resourceGroupName string, serverName string, failoverGroupName string) (armsql.FailoverGroupsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, failoverGroupName, nil)
}

// NewSqlFailoverGroupsClient creates a new SqlFailoverGroupsClient from the Azure SDK client
func NewSqlFailoverGroupsClient(client *armsql.FailoverGroupsClient) SqlFailoverGroupsClient {
	return &sqlFailoverGroupsClient{client: client}
}
