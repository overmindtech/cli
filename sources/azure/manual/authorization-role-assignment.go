package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var AuthorizationRoleAssignmentLookupByName = shared.NewItemTypeLookup("name", azureshared.AuthorizationRoleAssignment)

type authorizationRoleAssignmentWrapper struct {
	client clients.RoleAssignmentsClient

	*azureshared.ResourceGroupBase
}

func NewAuthorizationRoleAssignment(client clients.RoleAssignmentsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &authorizationRoleAssignmentWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.AuthorizationRoleAssignment,
		),
	}
}

func (a authorizationRoleAssignmentWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	if scope == "" {
		return nil, azureshared.QueryError(errors.New("scope cannot be empty"), scope, a.Type())
	}
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = a.ResourceGroup()
	}
	pager := a.client.ListForResourceGroup(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, a.Type())
		}
		for _, roleAssignment := range page.Value {
			item, sdpErr := a.azureRoleAssignmentToSDPItem(roleAssignment, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (a authorizationRoleAssignmentWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if scope == "" {
		return nil, azureshared.QueryError(errors.New("scope cannot be empty"), scope, a.Type())
	}
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: roleAssignmentName"), scope, a.Type())
	}

	roleAssignmentName := queryParts[0]
	if roleAssignmentName == "" {
		return nil, azureshared.QueryError(errors.New("roleAssignmentName cannot be empty"), scope, a.Type())
	}

	// Construct the Azure scope path from either subscription ID or resource group name
	azureScope := azureshared.ConstructRoleAssignmentScope(scope, a.SubscriptionID())
	if azureScope == "" {
		return nil, azureshared.QueryError(errors.New("failed to construct Azure scope path"), scope, a.Type())
	}

	resp, err := a.client.Get(ctx, azureScope, roleAssignmentName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, a.Type())
	}

	return a.azureRoleAssignmentToSDPItem(&resp.RoleAssignment, scope)
}

func (a authorizationRoleAssignmentWrapper) azureRoleAssignmentToSDPItem(roleAssignment *armauthorization.RoleAssignment, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(roleAssignment)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, a.Type())
	}

	// Extract role assignment name
	var roleAssignmentName string
	if roleAssignment.Name != nil {
		roleAssignmentName = *roleAssignment.Name
	}

	if roleAssignmentName == "" {
		return nil, azureshared.QueryError(errors.New("role assignment name cannot be empty"), scope, a.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = a.ResourceGroup()
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(resourceGroup, roleAssignmentName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, a.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.AuthorizationRoleAssignment.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to Delegated Managed Identity (external resource) if present
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
	if roleAssignment.Properties != nil && roleAssignment.Properties.DelegatedManagedIdentityResourceID != nil && *roleAssignment.Properties.DelegatedManagedIdentityResourceID != "" {
		identityResourceID := *roleAssignment.Properties.DelegatedManagedIdentityResourceID
		identityName := azureshared.ExtractResourceName(identityResourceID)
		if identityName != "" {
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  identityName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Role assignment depends on managed identity for delegated access
					// If identity is deleted/modified, role assignment operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Role Definition (external resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/get
	// GET /{scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}
	// Role definitions are subscription-level resources
	if roleAssignment.Properties != nil && roleAssignment.Properties.RoleDefinitionID != nil && *roleAssignment.Properties.RoleDefinitionID != "" {
		roleDefinitionID := *roleAssignment.Properties.RoleDefinitionID
		// Extract the role definition ID (GUID) from the full resource ID path
		// Format: /subscriptions/{subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}
		roleDefinitionGUID := azureshared.ExtractResourceName(roleDefinitionID)
		if roleDefinitionGUID != "" {
			// Extract subscription ID from the role definition ID path for scope
			// Role definitions are subscription-level, not resource group scoped
			linkedScope := azureshared.ExtractSubscriptionIDFromResourceID(roleDefinitionID)
			// Fallback: extract subscription ID from current scope if extraction failed
			if linkedScope == "" {
				scopeParts := strings.Split(scope, ".")
				if len(scopeParts) > 0 {
					linkedScope = scopeParts[0]
				}
			}
			if linkedScope != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.AuthorizationRoleDefinition.String(),
						Method: sdp.QueryMethod_GET,
						Query:  roleDefinitionGUID,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Role assignment depends on role definition for permissions
						// If role definition is deleted/modified, role assignment becomes invalid
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (a authorizationRoleAssignmentWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		AuthorizationRoleAssignmentLookupByName,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/role_assignment
func (a authorizationRoleAssignmentWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_role_assignment.id",
		},
	}
}

func (a authorizationRoleAssignmentWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ManagedIdentityUserAssignedIdentity,
		azureshared.AuthorizationRoleDefinition,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/management-and-governance#microsoftauthorization
func (a authorizationRoleAssignmentWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Authorization/roleAssignments/read",
	}
}

func (a authorizationRoleAssignmentWrapper) PredefinedRole() string {
	return "Reader"
}
