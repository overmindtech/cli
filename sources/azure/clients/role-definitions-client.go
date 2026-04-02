package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_role_definitions_client.go -package=mocks -source=role-definitions-client.go

// RoleDefinitionsPager is a type alias for the generic Pager interface with role definition response type.
type RoleDefinitionsPager = Pager[armauthorization.RoleDefinitionsClientListResponse]

// RoleDefinitionsClient is an interface for interacting with Azure role definitions
type RoleDefinitionsClient interface {
	NewListPager(scope string, options *armauthorization.RoleDefinitionsClientListOptions) RoleDefinitionsPager
	Get(ctx context.Context, scope string, roleDefinitionID string, options *armauthorization.RoleDefinitionsClientGetOptions) (armauthorization.RoleDefinitionsClientGetResponse, error)
}

type roleDefinitionsClient struct {
	client *armauthorization.RoleDefinitionsClient
}

func (c *roleDefinitionsClient) NewListPager(scope string, options *armauthorization.RoleDefinitionsClientListOptions) RoleDefinitionsPager {
	return c.client.NewListPager(scope, options)
}

func (c *roleDefinitionsClient) Get(ctx context.Context, scope string, roleDefinitionID string, options *armauthorization.RoleDefinitionsClientGetOptions) (armauthorization.RoleDefinitionsClientGetResponse, error) {
	return c.client.Get(ctx, scope, roleDefinitionID, options)
}

// NewRoleDefinitionsClient creates a new RoleDefinitionsClient from the Azure SDK client
func NewRoleDefinitionsClient(client *armauthorization.RoleDefinitionsClient) RoleDefinitionsClient {
	return &roleDefinitionsClient{client: client}
}
