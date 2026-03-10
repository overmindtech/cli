package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_default_security_rules_client.go -package=mocks -source=default-security-rules-client.go

// DefaultSecurityRulesPager is a type alias for the generic Pager interface with default security rules list response type.
type DefaultSecurityRulesPager = Pager[armnetwork.DefaultSecurityRulesClientListResponse]

// DefaultSecurityRulesClient is an interface for interacting with Azure NSG default security rules (child of network security group).
type DefaultSecurityRulesClient interface {
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, defaultSecurityRuleName string, options *armnetwork.DefaultSecurityRulesClientGetOptions) (armnetwork.DefaultSecurityRulesClientGetResponse, error)
	NewListPager(resourceGroupName string, networkSecurityGroupName string, options *armnetwork.DefaultSecurityRulesClientListOptions) DefaultSecurityRulesPager
}

type defaultSecurityRulesClient struct {
	client *armnetwork.DefaultSecurityRulesClient
}

func (c *defaultSecurityRulesClient) Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, defaultSecurityRuleName string, options *armnetwork.DefaultSecurityRulesClientGetOptions) (armnetwork.DefaultSecurityRulesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, networkSecurityGroupName, defaultSecurityRuleName, options)
}

func (c *defaultSecurityRulesClient) NewListPager(resourceGroupName string, networkSecurityGroupName string, options *armnetwork.DefaultSecurityRulesClientListOptions) DefaultSecurityRulesPager {
	return c.client.NewListPager(resourceGroupName, networkSecurityGroupName, options)
}

// NewDefaultSecurityRulesClient creates a new DefaultSecurityRulesClient from the Azure SDK client.
func NewDefaultSecurityRulesClient(client *armnetwork.DefaultSecurityRulesClient) DefaultSecurityRulesClient {
	return &defaultSecurityRulesClient{client: client}
}
