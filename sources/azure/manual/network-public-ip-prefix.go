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
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkPublicIPPrefixLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkPublicIPPrefix)

type networkPublicIPPrefixWrapper struct {
	client clients.PublicIPPrefixesClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkPublicIPPrefix(client clients.PublicIPPrefixesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkPublicIPPrefixWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkPublicIPPrefix,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-prefixes/list
func (n networkPublicIPPrefixWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, prefix := range page.Value {
			if prefix.Name == nil {
				continue
			}
			item, sdpErr := n.azurePublicIPPrefixToSDPItem(prefix, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkPublicIPPrefixWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
		for _, prefix := range page.Value {
			if prefix.Name == nil {
				continue
			}
			item, sdpErr := n.azurePublicIPPrefixToSDPItem(prefix, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-prefixes/get
func (n networkPublicIPPrefixWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part (public IP prefix name)"), scope, n.Type())
	}
	publicIPPrefixName := queryParts[0]
	if publicIPPrefixName == "" {
		return nil, azureshared.QueryError(errors.New("public IP prefix name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, publicIPPrefixName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azurePublicIPPrefixToSDPItem(&resp.PublicIPPrefix, scope)
}

func (n networkPublicIPPrefixWrapper) azurePublicIPPrefixToSDPItem(prefix *armnetwork.PublicIPPrefix, scope string) (*sdp.Item, *sdp.QueryError) {
	if prefix.Name == nil {
		return nil, azureshared.QueryError(errors.New("public IP prefix name is nil"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(prefix, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkPublicIPPrefix.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(prefix.Tags),
		LinkedItemQueries:  []*sdp.LinkedItemQuery{},
	}

	// Link to Custom Location when ExtendedLocation.Name is a custom location resource ID (Microsoft.ExtendedLocation/customLocations)
	if prefix.ExtendedLocation != nil && prefix.ExtendedLocation.Name != nil {
		customLocationID := *prefix.ExtendedLocation.Name
		if strings.Contains(customLocationID, "customLocations") {
			customLocationName := azureshared.ExtractResourceName(customLocationID)
			if customLocationName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(customLocationID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ExtendedLocationCustomLocation.String(),
						Method: sdp.QueryMethod_GET,
						Query:  customLocationName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	// Link to IP (standard library) for allocated prefix (e.g. "20.10.0.0/28")
	if prefix.Properties != nil && prefix.Properties.IPPrefix != nil && *prefix.Properties.IPPrefix != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  *prefix.Properties.IPPrefix,
				Scope:  "global",
			},
		})
	}

	if prefix.Properties != nil {
		// Link to Custom IP Prefix (parent prefix this prefix is associated with)
		if prefix.Properties.CustomIPPrefix != nil && prefix.Properties.CustomIPPrefix.ID != nil {
			customPrefixID := *prefix.Properties.CustomIPPrefix.ID
			customPrefixName := azureshared.ExtractResourceName(customPrefixID)
			if customPrefixName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(customPrefixID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkCustomIPPrefix.String(),
						Method: sdp.QueryMethod_GET,
						Query:  customPrefixName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to NAT Gateway
		if prefix.Properties.NatGateway != nil && prefix.Properties.NatGateway.ID != nil {
			natGatewayID := *prefix.Properties.NatGateway.ID
			natGatewayName := azureshared.ExtractResourceName(natGatewayID)
			if natGatewayName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(natGatewayID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkNatGateway.String(),
						Method: sdp.QueryMethod_GET,
						Query:  natGatewayName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to Load Balancer and Frontend IP Configuration (from frontend IP configuration reference)
		if prefix.Properties.LoadBalancerFrontendIPConfiguration != nil && prefix.Properties.LoadBalancerFrontendIPConfiguration.ID != nil {
			feConfigID := *prefix.Properties.LoadBalancerFrontendIPConfiguration.ID
			// Format: .../loadBalancers/{lbName}/frontendIPConfigurations/{feConfigName}
			params := azureshared.ExtractPathParamsFromResourceID(feConfigID, []string{"loadBalancers", "frontendIPConfigurations"})
			if len(params) >= 2 && params[0] != "" && params[1] != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(feConfigID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancer.String(),
						Method: sdp.QueryMethod_GET,
						Query:  params[0],
						Scope:  linkedScope,
					},
				})
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(params[0], params[1]),
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to each referenced Public IP Address
		for _, ref := range prefix.Properties.PublicIPAddresses {
			if ref != nil && ref.ID != nil {
				refID := *ref.ID
				refName := azureshared.ExtractResourceName(refID)
				if refName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(refID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPAddress.String(),
							Method: sdp.QueryMethod_GET,
							Query:  refName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Health from provisioning state
	if prefix.Properties != nil && prefix.Properties.ProvisioningState != nil {
		switch *prefix.Properties.ProvisioningState {
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

func (n networkPublicIPPrefixWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPublicIPPrefixLookupByName,
	}
}

func (n networkPublicIPPrefixWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkCustomIPPrefix:                      true,
		azureshared.NetworkNatGateway:                          true,
		azureshared.NetworkLoadBalancer:                        true,
		azureshared.NetworkLoadBalancerFrontendIPConfiguration: true,
		azureshared.NetworkPublicIPAddress:                     true,
		azureshared.ExtendedLocationCustomLocation:             true,
		stdlib.NetworkIP:                                       true,
	}
}

func (n networkPublicIPPrefixWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_public_ip_prefix.name",
		},
	}
}

// https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkPublicIPPrefixWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/publicIPPrefixes/read",
	}
}

func (n networkPublicIPPrefixWrapper) PredefinedRole() string {
	return "Reader"
}
