package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_server_virtual_network_rule_client.go -package=mocks -source=sql-server-virtual-network-rule-client.go

// SqlServerVirtualNetworkRulePager is a type alias for the generic Pager interface with SQL server virtual network rule list response type.
type SqlServerVirtualNetworkRulePager = Pager[armsql.VirtualNetworkRulesClientListByServerResponse]

// SqlServerVirtualNetworkRuleClient is an interface for interacting with Azure SQL server virtual network rules.
type SqlServerVirtualNetworkRuleClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlServerVirtualNetworkRulePager
	Get(ctx context.Context, resourceGroupName string, serverName string, virtualNetworkRuleName string) (armsql.VirtualNetworkRulesClientGetResponse, error)
}

type sqlServerVirtualNetworkRuleClient struct {
	client *armsql.VirtualNetworkRulesClient
}

func (a *sqlServerVirtualNetworkRuleClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlServerVirtualNetworkRulePager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlServerVirtualNetworkRuleClient) Get(ctx context.Context, resourceGroupName string, serverName string, virtualNetworkRuleName string) (armsql.VirtualNetworkRulesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, virtualNetworkRuleName, nil)
}

// NewSqlServerVirtualNetworkRuleClient creates a new SqlServerVirtualNetworkRuleClient from the Azure SDK client.
func NewSqlServerVirtualNetworkRuleClient(client *armsql.VirtualNetworkRulesClient) SqlServerVirtualNetworkRuleClient {
	return &sqlServerVirtualNetworkRuleClient{client: client}
}
