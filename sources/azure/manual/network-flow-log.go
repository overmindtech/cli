package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	NetworkWatcherLookupByName       = shared.NewItemTypeLookup("name", azureshared.NetworkNetworkWatcher)
	NetworkFlowLogLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkFlowLog)
)

type networkFlowLogWrapper struct {
	client clients.FlowLogsClient
	*azureshared.MultiResourceGroupBase
}

// NewNetworkFlowLog creates a new networkFlowLogWrapper instance (SearchableWrapper: child of network watcher).
func NewNetworkFlowLog(client clients.FlowLogsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkFlowLogWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkFlowLog,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-watcher/flow-logs/get
func (c networkFlowLogWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: networkWatcherName and flowLogName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	networkWatcherName := queryParts[0]
	flowLogName := queryParts[1]
	if networkWatcherName == "" {
		return nil, azureshared.QueryError(errors.New("networkWatcherName cannot be empty"), scope, c.Type())
	}
	if flowLogName == "" {
		return nil, azureshared.QueryError(errors.New("flowLogName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, networkWatcherName, flowLogName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureFlowLogToSDPItem(&resp.FlowLog, networkWatcherName, flowLogName, scope)
}

func (c networkFlowLogWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkWatcherLookupByName,
		NetworkFlowLogLookupByUniqueAttr,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-watcher/flow-logs/list
func (c networkFlowLogWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: networkWatcherName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	networkWatcherName := queryParts[0]

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, networkWatcherName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, flowLog := range page.Value {
			if flowLog == nil || flowLog.Name == nil {
				continue
			}
			item, sdpErr := c.azureFlowLogToSDPItem(flowLog, networkWatcherName, *flowLog.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c networkFlowLogWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: networkWatcherName"), scope, c.Type()))
		return
	}
	networkWatcherName := queryParts[0]

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, networkWatcherName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, flowLog := range page.Value {
			if flowLog == nil || flowLog.Name == nil {
				continue
			}
			item, sdpErr := c.azureFlowLogToSDPItem(flowLog, networkWatcherName, *flowLog.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c networkFlowLogWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{NetworkWatcherLookupByName},
	}
}

func (c networkFlowLogWrapper) azureFlowLogToSDPItem(flowLog *armnetwork.FlowLog, networkWatcherName, flowLogName, scope string) (*sdp.Item, *sdp.QueryError) {
	if flowLog.Name == nil {
		return nil, azureshared.QueryError(errors.New("resource name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(flowLog, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(networkWatcherName, flowLogName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkFlowLog.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(flowLog.Tags),
	}

	if flowLog.Properties != nil {
		// Health mapping from ProvisioningState
		if flowLog.Properties.ProvisioningState != nil {
			switch *flowLog.Properties.ProvisioningState {
			case armnetwork.ProvisioningStateSucceeded:
				sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
			case armnetwork.ProvisioningStateCreating, armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting:
				sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
			case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
				sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
			default:
				sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			}
		}

		// Link to TargetResourceID (polymorphic: NSG, VNet, or Subnet)
		if flowLog.Properties.TargetResourceID != nil && *flowLog.Properties.TargetResourceID != "" {
			targetID := *flowLog.Properties.TargetResourceID
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(targetID); extractedScope != "" {
				linkedScope = extractedScope
			}

			switch {
			case strings.Contains(targetID, "/networkSecurityGroups/"):
				nsgName := azureshared.ExtractResourceName(targetID)
				if nsgName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkSecurityGroup.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nsgName,
							Scope:  linkedScope,
						},
					})
				}
			case strings.Contains(targetID, "/subnets/"):
				params := azureshared.ExtractPathParamsFromResourceID(targetID, []string{"virtualNetworks", "subnets"})
				if len(params) >= 2 {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			case strings.Contains(targetID, "/virtualNetworks/"):
				vnetName := azureshared.ExtractResourceName(targetID)
				if vnetName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vnetName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to StorageID (storage account)
		if flowLog.Properties.StorageID != nil && *flowLog.Properties.StorageID != "" {
			storageAccountName := azureshared.ExtractResourceName(*flowLog.Properties.StorageID)
			if storageAccountName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*flowLog.Properties.StorageID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.StorageAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  storageAccountName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to Traffic Analytics workspace
		if flowLog.Properties.FlowAnalyticsConfiguration != nil &&
			flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration != nil &&
			flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID != nil &&
			*flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID != "" {
			workspaceName := azureshared.ExtractResourceName(*flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID)
			if workspaceName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.OperationalInsightsWorkspace.String(),
						Method: sdp.QueryMethod_GET,
						Query:  workspaceName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	// Link to parent NetworkWatcher
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkNetworkWatcher.String(),
			Method: sdp.QueryMethod_GET,
			Query:  networkWatcherName,
			Scope:  scope,
		},
	})

	// Link to user-assigned managed identities
	if flowLog.Identity != nil && flowLog.Identity.UserAssignedIdentities != nil {
		for identityID := range flowLog.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityID)
			if identityName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (c networkFlowLogWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkNetworkWatcher,
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkSubnet,
		azureshared.StorageAccount,
		azureshared.OperationalInsightsWorkspace,
		azureshared.ManagedIdentityUserAssignedIdentity,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkFlowLogWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkWatchers/flowLogs/read",
	}
}

func (c networkFlowLogWrapper) PredefinedRole() string {
	return "Reader"
}
