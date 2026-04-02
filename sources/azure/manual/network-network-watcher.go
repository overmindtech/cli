package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var NetworkNetworkWatcherLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkNetworkWatcher)

type networkNetworkWatcherWrapper struct {
	client clients.NetworkWatchersClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkNetworkWatcher creates a new NetworkNetworkWatcher adapter (ListableWrapper: top-level resource).
func NewNetworkNetworkWatcher(client clients.NetworkWatchersClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkNetworkWatcherWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkNetworkWatcher,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-watcher/network-watchers/list
func (c networkNetworkWatcherWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, watcher := range page.Value {
			if watcher.Name == nil {
				continue
			}
			item, sdpErr := c.azureNetworkWatcherToSDPItem(watcher, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c networkNetworkWatcherWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, watcher := range page.Value {
			if watcher.Name == nil {
				continue
			}
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureNetworkWatcherToSDPItem(watcher, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-watcher/network-watchers/get
func (c networkNetworkWatcherWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the network watcher name"), scope, c.Type())
	}
	networkWatcherName := queryParts[0]
	if networkWatcherName == "" {
		return nil, azureshared.QueryError(errors.New("networkWatcherName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	result, err := c.client.Get(ctx, rgScope.ResourceGroup, networkWatcherName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureNetworkWatcherToSDPItem(&result.Watcher, scope)
}

func (c networkNetworkWatcherWrapper) azureNetworkWatcherToSDPItem(watcher *armnetwork.Watcher, scope string) (*sdp.Item, *sdp.QueryError) {
	if watcher.Name == nil {
		return nil, azureshared.QueryError(errors.New("network watcher name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(watcher, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkNetworkWatcher.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(watcher.Tags),
	}

	// Map provisioning state to health
	if watcher.Properties != nil && watcher.Properties.ProvisioningState != nil {
		switch *watcher.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting, armnetwork.ProvisioningStateCreating:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to child FlowLogs via SEARCH
	// FlowLogs are child resources of NetworkWatcher, so we link via SEARCH with the network watcher name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkFlowLog.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  *watcher.Name,
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (c networkNetworkWatcherWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNetworkWatcherLookupByName,
	}
}

func (c networkNetworkWatcherWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkFlowLog,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkNetworkWatcherWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkWatchers/read",
	}
}

func (c networkNetworkWatcherWrapper) PredefinedRole() string {
	return "Reader"
}
