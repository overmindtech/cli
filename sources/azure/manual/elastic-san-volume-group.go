package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type elasticSanVolumeGroupWrapper struct {
	client clients.ElasticSanVolumeGroupClient
	*azureshared.MultiResourceGroupBase
}

func NewElasticSanVolumeGroup(client clients.ElasticSanVolumeGroupClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &elasticSanVolumeGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ElasticSanVolumeGroup,
		),
	}
}

func (e elasticSanVolumeGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Get requires 2 query parts: elasticSanName and volumeGroupName"), scope, e.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type())
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		return nil, azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, e.Type())
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	resp, err := e.client.Get(ctx, rgScope.ResourceGroup, elasticSanName, volumeGroupName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	return e.azureVolumeGroupToSDPItem(&resp.VolumeGroup, elasticSanName, volumeGroupName, scope)
}

func (e elasticSanVolumeGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ElasticSanLookupByName,
		ElasticSanVolumeGroupLookupByName,
	}
}

func (e elasticSanVolumeGroupWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Search requires 1 query part: elasticSanName"), scope, e.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type())
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	pager := e.client.NewListByElasticSanPager(rgScope.ResourceGroup, elasticSanName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, e.Type())
		}
		for _, vg := range page.Value {
			if vg.Name == nil {
				continue
			}
			item, sdpErr := e.azureVolumeGroupToSDPItem(vg, elasticSanName, *vg.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (e elasticSanVolumeGroupWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: elasticSanName"), scope, e.Type()))
		return
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		stream.SendError(azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type()))
		return
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, e.Type()))
		return
	}
	pager := e.client.NewListByElasticSanPager(rgScope.ResourceGroup, elasticSanName, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, e.Type()))
			return
		}
		for _, vg := range page.Value {
			if vg.Name == nil {
				continue
			}
			item, sdpErr := e.azureVolumeGroupToSDPItem(vg, elasticSanName, *vg.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (e elasticSanVolumeGroupWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{ElasticSanLookupByName},
	}
}

func (e elasticSanVolumeGroupWrapper) azureVolumeGroupToSDPItem(vg *armelasticsan.VolumeGroup, elasticSanName, volumeGroupName, scope string) (*sdp.Item, *sdp.QueryError) {
	if vg.Name == nil {
		return nil, azureshared.QueryError(errors.New("volume group name is nil"), scope, e.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(vg, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(elasticSanName, volumeGroupName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}

	item := &sdp.Item{
		Type:              azureshared.ElasticSanVolumeGroup.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to parent Elastic SAN
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSan.String(),
			Method: sdp.QueryMethod_GET,
			Query:  elasticSanName,
			Scope:  scope,
		},
	})

	// Link to User Assigned Identities from top-level Identity (map keys are ARM resource IDs)
	if vg.Identity != nil && vg.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range vg.Identity.UserAssignedIdentities {
			if identityResourceID == "" {
				continue
			}
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
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

	// Link to Private Endpoints via PrivateEndpointConnections
	if vg.Properties != nil && vg.Properties.PrivateEndpointConnections != nil {
		for _, pec := range vg.Properties.PrivateEndpointConnections {
			if pec != nil && pec.Properties != nil && pec.Properties.PrivateEndpoint != nil && pec.Properties.PrivateEndpoint.ID != nil {
				peName := azureshared.ExtractResourceName(*pec.Properties.PrivateEndpoint.ID)
				if peName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*pec.Properties.PrivateEndpoint.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  peName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Link to child Volume Snapshots (SEARCH by parent Elastic SAN + Volume Group)
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSanVolumeSnapshot.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName),
			Scope:  scope,
		},
	})

	// Link to child Volumes (SEARCH by parent Elastic SAN + Volume Group)
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSanVolume.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName),
			Scope:  scope,
		},
	})

	// Link to subnets from NetworkACLs virtual network rules
	if vg.Properties != nil && vg.Properties.NetworkACLs != nil && vg.Properties.NetworkACLs.VirtualNetworkRules != nil {
		for _, rule := range vg.Properties.NetworkACLs.VirtualNetworkRules {
			if rule != nil && rule.VirtualNetworkResourceID != nil && *rule.VirtualNetworkResourceID != "" {
				subnetID := *rule.VirtualNetworkResourceID
				params := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(subnetID)
					if linkedScope == "" {
						linkedScope = scope
					}
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Link to Key Vault and encryption identity from EncryptionProperties
	if vg.Properties != nil && vg.Properties.EncryptionProperties != nil {
		enc := vg.Properties.EncryptionProperties
		// Link to User Assigned Identity used for encryption (same pattern as storage-account.go)
		if enc.EncryptionIdentity != nil && enc.EncryptionIdentity.EncryptionUserAssignedIdentity != nil {
			identityResourceID := *enc.EncryptionIdentity.EncryptionUserAssignedIdentity
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
				})
			}
		}
		// Link to Key Vault and DNS from KeyVaultURI (DNS-resolvable hostname)
		if enc.KeyVaultProperties != nil && enc.KeyVaultProperties.KeyVaultURI != nil && *enc.KeyVaultProperties.KeyVaultURI != "" {
			keyVaultURI := *enc.KeyVaultProperties.KeyVaultURI
			vaultName := azureshared.ExtractVaultNameFromURI(keyVaultURI)
			if vaultName != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vaultName,
						Scope:  scope, // Key Vault URI does not contain resource group
					},
				})
			}
			if dnsName := azureshared.ExtractDNSFromURL(keyVaultURI); dnsName != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  dnsName,
						Scope:  "global",
					},
				})
			}
		}
	}

	// Health from provisioning state
	if vg.Properties != nil && vg.Properties.ProvisioningState != nil {
		switch *vg.Properties.ProvisioningState {
		case armelasticsan.ProvisioningStatesSucceeded:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case armelasticsan.ProvisioningStatesCreating, armelasticsan.ProvisioningStatesUpdating, armelasticsan.ProvisioningStatesDeleting,
			armelasticsan.ProvisioningStatesPending, armelasticsan.ProvisioningStatesRestoring:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armelasticsan.ProvisioningStatesFailed, armelasticsan.ProvisioningStatesCanceled,
			armelasticsan.ProvisioningStatesDeleted, armelasticsan.ProvisioningStatesInvalid:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return item, nil
}

func (e elasticSanVolumeGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ElasticSan:                          true,
		azureshared.ElasticSanVolume:                    true,
		azureshared.ElasticSanVolumeSnapshot:            true,
		azureshared.NetworkPrivateEndpoint:              true,
		azureshared.NetworkSubnet:                       true,
		azureshared.KeyVaultVault:                       true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		stdlib.NetworkDNS:                               true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/elastic_san_volume_group
func (e elasticSanVolumeGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_elastic_san_volume_group.id",
		},
	}
}

func (e elasticSanVolumeGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ElasticSan/elasticSans/volumegroups/read",
	}
}

func (e elasticSanVolumeGroupWrapper) PredefinedRole() string {
	return "Reader"
}
