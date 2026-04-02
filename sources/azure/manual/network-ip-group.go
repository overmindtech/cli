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
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkIPGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkIPGroup)

type networkIPGroupWrapper struct {
	client clients.IPGroupsClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkIPGroup creates a new networkIPGroupWrapper instance.
func NewNetworkIPGroup(client clients.IPGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkIPGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkIPGroup,
		),
	}
}

// List retrieves all IP groups in a resource group.
// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ip-groups/list-by-resource-group
func (c networkIPGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, ipGroup := range page.Value {
			if ipGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureIPGroupToSDPItem(ipGroup, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

// ListStream streams all IP groups in a resource group.
func (c networkIPGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
		for _, ipGroup := range page.Value {
			if ipGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureIPGroupToSDPItem(ipGroup, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// Get retrieves a single IP group by name.
// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ip-groups/get
func (c networkIPGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part (IP group name)"), scope, c.Type())
	}
	ipGroupName := queryParts[0]
	if ipGroupName == "" {
		return nil, azureshared.QueryError(errors.New("IP group name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, ipGroupName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureIPGroupToSDPItem(&resp.IPGroup, scope)
}

func (c networkIPGroupWrapper) azureIPGroupToSDPItem(ipGroup *armnetwork.IPGroup, scope string) (*sdp.Item, *sdp.QueryError) {
	if ipGroup.Name == nil {
		return nil, azureshared.QueryError(errors.New("IP group name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(ipGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkIPGroup.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(ipGroup.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health from provisioning state
	if ipGroup.Properties != nil && ipGroup.Properties.ProvisioningState != nil {
		switch *ipGroup.Properties.ProvisioningState {
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

	if ipGroup.Properties != nil {
		// Link to IP addresses
		// IP Groups contain a list of IP addresses or prefixes
		for _, ipAddr := range ipGroup.Properties.IPAddresses {
			if ipAddr != nil && *ipAddr != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *ipAddr,
						Scope:  "global",
					},
				})
			}
		}

		// Link to Firewalls (read-only, references back to Azure Firewalls using this IP Group)
		// Note: These are SubResource references containing just IDs
		for _, firewall := range ipGroup.Properties.Firewalls {
			if firewall != nil && firewall.ID != nil {
				firewallName := azureshared.ExtractResourceName(*firewall.ID)
				if firewallName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*firewall.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkFirewall.String(),
							Method: sdp.QueryMethod_GET,
							Query:  firewallName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Firewall Policies (read-only, references back to Firewall Policies using this IP Group)
		for _, policy := range ipGroup.Properties.FirewallPolicies {
			if policy != nil && policy.ID != nil {
				policyName := azureshared.ExtractResourceName(*policy.ID)
				if policyName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*policy.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkFirewallPolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  policyName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c networkIPGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkIPGroupLookupByName,
	}
}

func (c networkIPGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		stdlib.NetworkIP:                  true,
		azureshared.NetworkFirewall:       true,
		azureshared.NetworkFirewallPolicy: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkIPGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/ipGroups/read",
	}
}

func (c networkIPGroupWrapper) PredefinedRole() string {
	return "Reader"
}
