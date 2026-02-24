package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_sql_server_firewall_rule_client.go -package=mocks -source=sql-server-firewall-rule-client.go

// SqlServerFirewallRulePager is a type alias for the generic Pager interface with SQL server firewall rule response type.
type SqlServerFirewallRulePager = Pager[armsql.FirewallRulesClientListByServerResponse]

// SqlServerFirewallRuleClient is an interface for interacting with Azure SQL server firewall rules.
type SqlServerFirewallRuleClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlServerFirewallRulePager
	Get(ctx context.Context, resourceGroupName string, serverName string, firewallRuleName string) (armsql.FirewallRulesClientGetResponse, error)
}

type sqlServerFirewallRuleClient struct {
	client *armsql.FirewallRulesClient
}

func (a *sqlServerFirewallRuleClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) SqlServerFirewallRulePager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *sqlServerFirewallRuleClient) Get(ctx context.Context, resourceGroupName string, serverName string, firewallRuleName string) (armsql.FirewallRulesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, firewallRuleName, nil)
}

// NewSqlServerFirewallRuleClient creates a new SqlServerFirewallRuleClient from the Azure SDK client.
func NewSqlServerFirewallRuleClient(client *armsql.FirewallRulesClient) SqlServerFirewallRuleClient {
	return &sqlServerFirewallRuleClient{client: client}
}
