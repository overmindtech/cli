package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_role_assignments_client.go -package=mocks -source=role-assignments-client.go

// RoleAssignmentsPager is a type alias for the generic Pager interface with role assignment response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type RoleAssignmentsPager = Pager[armauthorization.RoleAssignmentsClientListForResourceGroupResponse]

// RoleAssignmentsClient is an interface for interacting with Azure role assignments
type RoleAssignmentsClient interface {
	ListForResourceGroup(resourceGroupName string, options *armauthorization.RoleAssignmentsClientListForResourceGroupOptions) RoleAssignmentsPager
	Get(ctx context.Context, scope string, roleAssignmentName string, options *armauthorization.RoleAssignmentsClientGetOptions) (armauthorization.RoleAssignmentsClientGetResponse, error)
}

type roleAssignmentsClient struct {
	client *armauthorization.RoleAssignmentsClient
}

func (c *roleAssignmentsClient) ListForResourceGroup(resourceGroupName string, options *armauthorization.RoleAssignmentsClientListForResourceGroupOptions) RoleAssignmentsPager {
	return c.client.NewListForResourceGroupPager(resourceGroupName, options)
}

func (c *roleAssignmentsClient) Get(ctx context.Context, scope string, roleAssignmentName string, options *armauthorization.RoleAssignmentsClientGetOptions) (armauthorization.RoleAssignmentsClientGetResponse, error) {
	return c.client.Get(ctx, scope, roleAssignmentName, options)
}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient from the Azure SDK client
func NewRoleAssignmentsClient(client *armauthorization.RoleAssignmentsClient) RoleAssignmentsClient {
	return &roleAssignmentsClient{client: client}
}
