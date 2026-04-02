package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var AuthorizationRoleDefinitionLookupByName = shared.NewItemTypeLookup("name", azureshared.AuthorizationRoleDefinition)

type authorizationRoleDefinitionWrapper struct {
	client clients.RoleDefinitionsClient

	*azureshared.SubscriptionBase
}

func NewAuthorizationRoleDefinition(client clients.RoleDefinitionsClient, subscriptionID string) sources.ListableWrapper {
	return &authorizationRoleDefinitionWrapper{
		client: client,
		SubscriptionBase: azureshared.NewSubscriptionBase(
			subscriptionID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.AuthorizationRoleDefinition,
		),
	}
}

// List retrieves all role definitions within the subscription scope.
// ref: https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list
func (c authorizationRoleDefinitionWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	if scope == "" {
		return nil, azureshared.QueryError(errors.New("scope cannot be empty"), scope, c.Type())
	}

	azureScope := fmt.Sprintf("/subscriptions/%s", c.SubscriptionID())
	pager := c.client.NewListPager(azureScope, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, roleDefinition := range page.Value {
			if roleDefinition == nil || roleDefinition.Name == nil {
				continue
			}
			item, sdpErr := c.azureRoleDefinitionToSDPItem(roleDefinition, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// ListStream streams all role definitions within the subscription scope.
func (c authorizationRoleDefinitionWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	if scope == "" {
		stream.SendError(azureshared.QueryError(errors.New("scope cannot be empty"), scope, c.Type()))
		return
	}

	azureScope := fmt.Sprintf("/subscriptions/%s", c.SubscriptionID())
	pager := c.client.NewListPager(azureScope, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, roleDefinition := range page.Value {
			if roleDefinition == nil || roleDefinition.Name == nil {
				continue
			}
			item, sdpErr := c.azureRoleDefinitionToSDPItem(roleDefinition, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// Get retrieves a role definition by its ID (GUID).
// ref: https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/get
func (c authorizationRoleDefinitionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if scope == "" {
		return nil, azureshared.QueryError(errors.New("scope cannot be empty"), scope, c.Type())
	}
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: roleDefinitionID"), scope, c.Type())
	}

	roleDefinitionID := queryParts[0]
	if roleDefinitionID == "" {
		return nil, azureshared.QueryError(errors.New("roleDefinitionID cannot be empty"), scope, c.Type())
	}

	azureScope := fmt.Sprintf("/subscriptions/%s", c.SubscriptionID())
	resp, err := c.client.Get(ctx, azureScope, roleDefinitionID, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureRoleDefinitionToSDPItem(&resp.RoleDefinition, scope)
}

func (c authorizationRoleDefinitionWrapper) azureRoleDefinitionToSDPItem(roleDefinition *armauthorization.RoleDefinition, scope string) (*sdp.Item, *sdp.QueryError) {
	if roleDefinition.Name == nil {
		return nil, azureshared.QueryError(errors.New("role definition name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(roleDefinition)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.AuthorizationRoleDefinition.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to AssignableScopes (subscriptions and resource groups)
	if roleDefinition.Properties != nil && roleDefinition.Properties.AssignableScopes != nil {
		for _, assignableScope := range roleDefinition.Properties.AssignableScopes {
			if assignableScope == nil || *assignableScope == "" {
				continue
			}
			scopePath := *assignableScope

			// Determine if this is a subscription or resource group scope
			// Format: /subscriptions/{subscriptionId} or /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}
			if rgName := azureshared.ExtractResourceGroupFromResourceID(scopePath); rgName != "" {
				// Resource group scope
				subscriptionID := azureshared.ExtractSubscriptionIDFromResourceID(scopePath)
				if subscriptionID != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ResourcesResourceGroup.String(),
							Method: sdp.QueryMethod_GET,
							Query:  rgName,
							Scope:  subscriptionID,
						},
					})
				}
			} else if subscriptionID := azureshared.ExtractSubscriptionIDFromResourceID(scopePath); subscriptionID != "" {
				// Subscription scope only
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ResourcesSubscription.String(),
						Method: sdp.QueryMethod_GET,
						Query:  subscriptionID,
						Scope:  "global",
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (c authorizationRoleDefinitionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		AuthorizationRoleDefinitionLookupByName,
	}
}

// PotentialLinks returns all resource types this adapter can link to.
func (c authorizationRoleDefinitionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ResourcesSubscription,
		azureshared.ResourcesResourceGroup,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/management-and-governance#microsoftauthorization
func (c authorizationRoleDefinitionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Authorization/roleDefinitions/read",
	}
}

func (c authorizationRoleDefinitionWrapper) PredefinedRole() string {
	return "Reader"
}
