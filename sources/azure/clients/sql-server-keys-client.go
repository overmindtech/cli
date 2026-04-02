package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_server_keys_client.go -package=mocks -source=sql-server-keys-client.go

// SqlServerKeysPager is a type alias for the generic Pager interface with sql server keys response type.
type SqlServerKeysPager = Pager[armsql.ServerKeysClientListByServerResponse]

// SqlServerKeysClient is an interface for interacting with Azure SQL server keys
type SqlServerKeysClient interface {
	NewListByServerPager(resourceGroupName string, serverName string) SqlServerKeysPager
	Get(ctx context.Context, resourceGroupName string, serverName string, keyName string) (armsql.ServerKeysClientGetResponse, error)
}

type sqlServerKeysClient struct {
	client *armsql.ServerKeysClient
}

func (a *sqlServerKeysClient) NewListByServerPager(resourceGroupName string, serverName string) SqlServerKeysPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlServerKeysClient) Get(ctx context.Context, resourceGroupName string, serverName string, keyName string) (armsql.ServerKeysClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, keyName, nil)
}

// NewSqlServerKeysClient creates a new SqlServerKeysClient from the Azure SDK client
func NewSqlServerKeysClient(client *armsql.ServerKeysClient) SqlServerKeysClient {
	return &sqlServerKeysClient{client: client}
}
