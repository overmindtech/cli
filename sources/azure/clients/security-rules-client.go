package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_security_rules_client.go -package=mocks -source=security-rules-client.go

// SecurityRulesPager is a type alias for the generic Pager interface with security rules list response type.
type SecurityRulesPager = Pager[armnetwork.SecurityRulesClientListResponse]

// SecurityRulesClient is an interface for interacting with Azure NSG security rules (child of network security group).
type SecurityRulesClient interface {
	Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, securityRuleName string, options *armnetwork.SecurityRulesClientGetOptions) (armnetwork.SecurityRulesClientGetResponse, error)
	NewListPager(resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityRulesClientListOptions) SecurityRulesPager
}

type securityRulesClient struct {
	client *armnetwork.SecurityRulesClient
}

func (a *securityRulesClient) Get(ctx context.Context, resourceGroupName string, networkSecurityGroupName string, securityRuleName string, options *armnetwork.SecurityRulesClientGetOptions) (armnetwork.SecurityRulesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, networkSecurityGroupName, securityRuleName, options)
}

func (a *securityRulesClient) NewListPager(resourceGroupName string, networkSecurityGroupName string, options *armnetwork.SecurityRulesClientListOptions) SecurityRulesPager {
	return a.client.NewListPager(resourceGroupName, networkSecurityGroupName, options)
}

// NewSecurityRulesClient creates a new SecurityRulesClient from the Azure SDK client.
func NewSecurityRulesClient(client *armnetwork.SecurityRulesClient) SecurityRulesClient {
	return &securityRulesClient{client: client}
}
