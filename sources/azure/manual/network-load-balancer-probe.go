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

var NetworkLoadBalancerProbeLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkLoadBalancerProbe)

type networkLoadBalancerProbeWrapper struct {
	client clients.LoadBalancerProbesClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkLoadBalancerProbe(client clients.LoadBalancerProbesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkLoadBalancerProbeWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkLoadBalancerProbe,
		),
	}
}

func (c networkLoadBalancerProbeWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: loadBalancerName and probeName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]
	probeName := queryParts[1]

	if loadBalancerName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "loadBalancerName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	if probeName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "probeName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, loadBalancerName, probeName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureProbeToSDPItem(&resp.Probe, loadBalancerName, scope)
}

func (c networkLoadBalancerProbeWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: loadBalancerName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]

	if loadBalancerName == "" {
		return nil, azureshared.QueryError(errors.New("loadBalancerName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, loadBalancerName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}

		for _, probe := range page.Value {
			if probe == nil || probe.Name == nil {
				continue
			}
			item, sdpErr := c.azureProbeToSDPItem(probe, loadBalancerName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c networkLoadBalancerProbeWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: loadBalancerName"), scope, c.Type()))
		return
	}
	loadBalancerName := queryParts[0]

	if loadBalancerName == "" {
		stream.SendError(azureshared.QueryError(errors.New("loadBalancerName cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, loadBalancerName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, probe := range page.Value {
			if probe == nil || probe.Name == nil {
				continue
			}
			item, sdpErr := c.azureProbeToSDPItem(probe, loadBalancerName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c networkLoadBalancerProbeWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkLoadBalancerLookupByName,
		NetworkLoadBalancerProbeLookupByUniqueAttr,
	}
}

func (c networkLoadBalancerProbeWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkLoadBalancerLookupByName,
		},
	}
}

func (c networkLoadBalancerProbeWrapper) azureProbeToSDPItem(probe *armnetwork.Probe, loadBalancerName string, scope string) (*sdp.Item, *sdp.QueryError) {
	if probe.Name == nil {
		return nil, azureshared.QueryError(errors.New("probe name is nil"), scope, c.Type())
	}

	probeName := *probe.Name

	attributes, err := shared.ToAttributesWithExclude(probe, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(loadBalancerName, probeName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkLoadBalancerProbe.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	if probe.Properties != nil && probe.Properties.ProvisioningState != nil {
		switch *probe.Properties.ProvisioningState {
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

	// Link to parent Load Balancer
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkLoadBalancer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  loadBalancerName,
			Scope:  scope,
		},
	})

	if probe.Properties != nil {
		// Link to Load Balancing Rules that reference this probe
		for _, lbRule := range probe.Properties.LoadBalancingRules {
			if lbRule != nil && lbRule.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*lbRule.ID, []string{"loadBalancers", "loadBalancingRules"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*lbRule.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c networkLoadBalancerProbeWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkLoadBalancer:                  true,
		azureshared.NetworkLoadBalancerLoadBalancingRule: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (c networkLoadBalancerProbeWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/loadBalancers/probes/read",
	}
}

func (c networkLoadBalancerProbeWrapper) PredefinedRole() string {
	return "Reader"
}
