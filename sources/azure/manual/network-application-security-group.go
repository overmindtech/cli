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

var NetworkApplicationSecurityGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkApplicationSecurityGroup)

type networkApplicationSecurityGroupWrapper struct {
	client clients.ApplicationSecurityGroupsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkApplicationSecurityGroup(client clients.ApplicationSecurityGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkApplicationSecurityGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkApplicationSecurityGroup,
		),
	}
}

func (n networkApplicationSecurityGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, asg := range page.Value {
			if asg.Name == nil {
				continue
			}
			item, sdpErr := n.azureApplicationSecurityGroupToSDPItem(asg, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkApplicationSecurityGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, asg := range page.Value {
			if asg.Name == nil {
				continue
			}
			item, sdpErr := n.azureApplicationSecurityGroupToSDPItem(asg, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkApplicationSecurityGroupWrapper) azureApplicationSecurityGroupToSDPItem(asg *armnetwork.ApplicationSecurityGroup, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(asg, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	if asg.Name == nil {
		return nil, azureshared.QueryError(errors.New("application security group name is nil"), scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkApplicationSecurityGroup.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(asg.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// no links - https://learn.microsoft.com/en-us/rest/api/virtualnetwork/application-security-groups/get?view=rest-virtualnetwork-2025-05-01&tabs=HTTP

	// Health from provisioning state
	if asg.Properties != nil && asg.Properties.ProvisioningState != nil {
		switch *asg.Properties.ProvisioningState {
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

	return sdpItem, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/application-security-groups/get
func (n networkApplicationSecurityGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part (application security group name)"), scope, n.Type())
	}
	asgName := queryParts[0]
	if asgName == "" {
		return nil, azureshared.QueryError(errors.New("application security group name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, asgName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azureApplicationSecurityGroupToSDPItem(&resp.ApplicationSecurityGroup, scope)
}

func (n networkApplicationSecurityGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkApplicationSecurityGroupLookupByName,
	}
}

func (n networkApplicationSecurityGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/application_security_group
func (n networkApplicationSecurityGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_application_security_group.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkApplicationSecurityGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/applicationSecurityGroups/read",
	}
}

func (n networkApplicationSecurityGroupWrapper) PredefinedRole() string {
	return "Reader"
}
