package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

//go:generate mockgen -destination=../shared/mocks/mock_user_assigned_identities_client.go -package=mocks -source=user-assigned-identities-client.go

// UserAssignedIdentitiesPager is a type alias for the generic Pager interface with user assigned identity response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type UserAssignedIdentitiesPager = Pager[armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse]

// UserAssignedIdentitiesClient is an interface for interacting with Azure user assigned identities
type UserAssignedIdentitiesClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error)
	ListByResourceGroup(resourceGroupName string, options *armmsi.UserAssignedIdentitiesClientListByResourceGroupOptions) UserAssignedIdentitiesPager
}

type userAssignedIdentitiesClient struct {
	client *armmsi.UserAssignedIdentitiesClient
}

func (c *userAssignedIdentitiesClient) Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmsi.UserAssignedIdentitiesClientGetOptions) (armmsi.UserAssignedIdentitiesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, resourceName, options)
}

func (c *userAssignedIdentitiesClient) ListByResourceGroup(resourceGroupName string, options *armmsi.UserAssignedIdentitiesClientListByResourceGroupOptions) UserAssignedIdentitiesPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewUserAssignedIdentitiesClient creates a new UserAssignedIdentitiesClient from the Azure SDK client
func NewUserAssignedIdentitiesClient(client *armmsi.UserAssignedIdentitiesClient) UserAssignedIdentitiesClient {
	return &userAssignedIdentitiesClient{client: client}
}
