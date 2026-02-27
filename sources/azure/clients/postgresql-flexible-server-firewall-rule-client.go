package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_postgresql_flexible_server_firewall_rule_client.go -package=mocks -source=postgresql-flexible-server-firewall-rule-client.go

// PostgreSQLFlexibleServerFirewallRulePager is a type alias for the generic Pager interface with PostgreSQL flexible server firewall rule response type.
type PostgreSQLFlexibleServerFirewallRulePager = Pager[armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse]

// PostgreSQLFlexibleServerFirewallRuleClient is an interface for interacting with Azure PostgreSQL flexible server firewall rules.
type PostgreSQLFlexibleServerFirewallRuleClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) PostgreSQLFlexibleServerFirewallRulePager
	Get(ctx context.Context, resourceGroupName string, serverName string, firewallRuleName string) (armpostgresqlflexibleservers.FirewallRulesClientGetResponse, error)
}

type postgresqlFlexibleServerFirewallRuleClient struct {
	client *armpostgresqlflexibleservers.FirewallRulesClient
}

func (a *postgresqlFlexibleServerFirewallRuleClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) PostgreSQLFlexibleServerFirewallRulePager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *postgresqlFlexibleServerFirewallRuleClient) Get(ctx context.Context, resourceGroupName string, serverName string, firewallRuleName string) (armpostgresqlflexibleservers.FirewallRulesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, firewallRuleName, nil)
}

// NewPostgreSQLFlexibleServerFirewallRuleClient creates a new PostgreSQLFlexibleServerFirewallRuleClient from the Azure SDK client.
func NewPostgreSQLFlexibleServerFirewallRuleClient(client *armpostgresqlflexibleservers.FirewallRulesClient) PostgreSQLFlexibleServerFirewallRuleClient {
	return &postgresqlFlexibleServerFirewallRuleClient{client: client}
}
