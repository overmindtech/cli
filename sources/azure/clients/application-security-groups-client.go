package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_application_security_groups_client.go -package=mocks -source=application-security-groups-client.go

// ApplicationSecurityGroupsPager is a type alias for the generic Pager interface with application security group response type.
type ApplicationSecurityGroupsPager = Pager[armnetwork.ApplicationSecurityGroupsClientListResponse]

// ApplicationSecurityGroupsClient is an interface for interacting with Azure application security groups.
type ApplicationSecurityGroupsClient interface {
	Get(ctx context.Context, resourceGroupName string, applicationSecurityGroupName string, options *armnetwork.ApplicationSecurityGroupsClientGetOptions) (armnetwork.ApplicationSecurityGroupsClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armnetwork.ApplicationSecurityGroupsClientListOptions) ApplicationSecurityGroupsPager
}

type applicationSecurityGroupsClient struct {
	client *armnetwork.ApplicationSecurityGroupsClient
}

func (c *applicationSecurityGroupsClient) Get(ctx context.Context, resourceGroupName string, applicationSecurityGroupName string, options *armnetwork.ApplicationSecurityGroupsClientGetOptions) (armnetwork.ApplicationSecurityGroupsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, applicationSecurityGroupName, options)
}

func (c *applicationSecurityGroupsClient) NewListPager(resourceGroupName string, options *armnetwork.ApplicationSecurityGroupsClientListOptions) ApplicationSecurityGroupsPager {
	return c.client.NewListPager(resourceGroupName, options)
}

// NewApplicationSecurityGroupsClient creates a new ApplicationSecurityGroupsClient from the Azure SDK client.
func NewApplicationSecurityGroupsClient(client *armnetwork.ApplicationSecurityGroupsClient) ApplicationSecurityGroupsClient {
	return &applicationSecurityGroupsClient{client: client}
}
