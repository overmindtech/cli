package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var OperationalInsightsWorkspaceLookupByName = shared.NewItemTypeLookup("name", azureshared.OperationalInsightsWorkspace)

type operationalInsightsWorkspaceWrapper struct {
	client clients.OperationalInsightsWorkspaceClient

	*azureshared.MultiResourceGroupBase
}

func NewOperationalInsightsWorkspace(client clients.OperationalInsightsWorkspaceClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &operationalInsightsWorkspaceWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
			azureshared.OperationalInsightsWorkspace,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/list-by-resource-group
func (c operationalInsightsWorkspaceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, workspace := range page.Value {
			if workspace.Name == nil {
				continue
			}
			item, sdpErr := c.azureWorkspaceToSDPItem(workspace, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c operationalInsightsWorkspaceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, workspace := range page.Value {
			if workspace.Name == nil {
				continue
			}
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureWorkspaceToSDPItem(workspace, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/get
func (c operationalInsightsWorkspaceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the workspace name"), scope, c.Type())
	}
	workspaceName := queryParts[0]
	if workspaceName == "" {
		return nil, azureshared.QueryError(errors.New("workspaceName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	result, err := c.client.Get(ctx, rgScope.ResourceGroup, workspaceName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureWorkspaceToSDPItem(&result.Workspace, scope)
}

func (c operationalInsightsWorkspaceWrapper) azureWorkspaceToSDPItem(workspace *armoperationalinsights.Workspace, scope string) (*sdp.Item, *sdp.QueryError) {
	if workspace.Name == nil {
		return nil, azureshared.QueryError(errors.New("workspace name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(workspace, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.OperationalInsightsWorkspace.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(workspace.Tags),
	}

	// Health status mapping based on provisioning state
	if workspace.Properties != nil && workspace.Properties.ProvisioningState != nil {
		switch *workspace.Properties.ProvisioningState {
		case armoperationalinsights.WorkspaceEntityStatusSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armoperationalinsights.WorkspaceEntityStatusCreating,
			armoperationalinsights.WorkspaceEntityStatusUpdating,
			armoperationalinsights.WorkspaceEntityStatusDeleting,
			armoperationalinsights.WorkspaceEntityStatusProvisioningAccount:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armoperationalinsights.WorkspaceEntityStatusFailed,
			armoperationalinsights.WorkspaceEntityStatusCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to Private Link Scope Scoped Resources
	// PrivateLinkScopedResources[].ResourceID refers to Azure Monitor Private Link Scope
	// scoped resources (microsoft.insights/privateLinkScopes/scopedResources)
	if workspace.Properties != nil && workspace.Properties.PrivateLinkScopedResources != nil {
		for _, plsr := range workspace.Properties.PrivateLinkScopedResources {
			if plsr != nil && plsr.ResourceID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*plsr.ResourceID, []string{"privateLinkScopes", "scopedResources"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					scopeName, scopedResourceName := params[0], params[1]
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*plsr.ResourceID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.InsightsPrivateLinkScopeScopedResource.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(scopeName, scopedResourceName),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Link to Cluster (Dedicated Log Analytics cluster)
	if workspace.Properties != nil && workspace.Properties.Features != nil && workspace.Properties.Features.ClusterResourceID != nil {
		clusterName := azureshared.ExtractResourceName(*workspace.Properties.Features.ClusterResourceID)
		if clusterName != "" {
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(*workspace.Properties.Features.ClusterResourceID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.OperationalInsightsCluster.String(),
					Method: sdp.QueryMethod_GET,
					Query:  clusterName,
					Scope:  linkedScope,
				},
			})
		}
	}

	return sdpItem, nil
}

func (c operationalInsightsWorkspaceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		OperationalInsightsWorkspaceLookupByName,
	}
}

func (c operationalInsightsWorkspaceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.InsightsPrivateLinkScopeScopedResource,
		azureshared.OperationalInsightsCluster,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftoperationalinsights
func (c operationalInsightsWorkspaceWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.OperationalInsights/workspaces/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/monitor#log-analytics-reader
func (c operationalInsightsWorkspaceWrapper) PredefinedRole() string {
	return "Log Analytics Reader"
}
