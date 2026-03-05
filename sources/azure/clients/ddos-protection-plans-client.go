package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_ddos_protection_plans_client.go -package=mocks -source=ddos-protection-plans-client.go

// DdosProtectionPlansPager is a type alias for the generic Pager interface with DDoS protection plan list response type.
type DdosProtectionPlansPager = Pager[armnetwork.DdosProtectionPlansClientListByResourceGroupResponse]

// DdosProtectionPlansClient is an interface for interacting with Azure DDoS protection plans.
type DdosProtectionPlansClient interface {
	Get(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string, options *armnetwork.DdosProtectionPlansClientGetOptions) (armnetwork.DdosProtectionPlansClientGetResponse, error)
	NewListByResourceGroupPager(resourceGroupName string, options *armnetwork.DdosProtectionPlansClientListByResourceGroupOptions) DdosProtectionPlansPager
}

type ddosProtectionPlansClient struct {
	client *armnetwork.DdosProtectionPlansClient
}

func (c *ddosProtectionPlansClient) Get(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string, options *armnetwork.DdosProtectionPlansClientGetOptions) (armnetwork.DdosProtectionPlansClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, ddosProtectionPlanName, options)
}

func (c *ddosProtectionPlansClient) NewListByResourceGroupPager(resourceGroupName string, options *armnetwork.DdosProtectionPlansClientListByResourceGroupOptions) DdosProtectionPlansPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewDdosProtectionPlansClient creates a new DdosProtectionPlansClient from the Azure SDK client.
func NewDdosProtectionPlansClient(client *armnetwork.DdosProtectionPlansClient) DdosProtectionPlansClient {
	return &ddosProtectionPlansClient{client: client}
}
