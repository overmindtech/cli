package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeDiskEncryptionSetLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDiskEncryptionSet)

type computeDiskEncryptionSetWrapper struct {
	client clients.DiskEncryptionSetsClient
	*azureshared.ResourceGroupBase
}

func NewComputeDiskEncryptionSet(client clients.DiskEncryptionSetsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &computeDiskEncryptionSetWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ComputeDiskEncryptionSet,
		),
	}
}

func (c computeDiskEncryptionSetWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	pager := c.client.NewListByResourceGroupPager(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, diskEncryptionSet := range page.Value {
			if diskEncryptionSet.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskEncryptionSetToSDPItem(diskEncryptionSet, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeDiskEncryptionSetWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	pager := c.client.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, diskEncryptionSet := range page.Value {
			if diskEncryptionSet.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskEncryptionSetToSDPItem(diskEncryptionSet, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			stream.SendItem(item)
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
	}
}

func (c computeDiskEncryptionSetWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the disk encryption set name"), scope, c.Type())
	}
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	diskEncryptionSetName := queryParts[0]
	if diskEncryptionSetName == "" {
		return nil, azureshared.QueryError(errors.New("diskEncryptionSetName cannot be empty"), scope, c.Type())
	}
	diskEncryptionSet, err := c.client.Get(ctx, resourceGroup, diskEncryptionSetName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureDiskEncryptionSetToSDPItem(&diskEncryptionSet.DiskEncryptionSet, scope)
}

func (c computeDiskEncryptionSetWrapper) azureDiskEncryptionSetToSDPItem(diskEncryptionSet *armcompute.DiskEncryptionSet, scope string) (*sdp.Item, *sdp.QueryError) {
	if diskEncryptionSet.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(diskEncryptionSet, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeDiskEncryptionSet.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(diskEncryptionSet.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	hasLinkedQuery := func(itemType string, method sdp.QueryMethod, query string) bool {
		for _, liq := range sdpItem.GetLinkedItemQueries() {
			q := liq.GetQuery()
			if q == nil {
				continue
			}
			if q.GetType() == itemType && q.GetMethod() == method && q.GetQuery() == query {
				return true
			}
		}
		return false
	}

	// Link to Key Vault from Properties.ActiveKey.SourceVault.ID
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01
	if diskEncryptionSet.Properties != nil &&
		diskEncryptionSet.Properties.ActiveKey != nil &&
		diskEncryptionSet.Properties.ActiveKey.SourceVault != nil &&
		diskEncryptionSet.Properties.ActiveKey.SourceVault.ID != nil &&
		*diskEncryptionSet.Properties.ActiveKey.SourceVault.ID != "" {
		vaultID := *diskEncryptionSet.Properties.ActiveKey.SourceVault.ID
		vaultName := azureshared.ExtractResourceName(vaultID)
		if vaultName != "" {
			extractedScope := azureshared.ExtractScopeFromResourceID(vaultID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Key Vault is deleted/locked down → DES can't access key material (In: true)
					Out: false, // If DES is deleted/modified → Key Vault remains (Out: false)
				},
			})
		}
	}

	// Link to Key Vault(s) from Properties.PreviousKeys[].SourceVault.ID
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01
	if diskEncryptionSet.Properties != nil && len(diskEncryptionSet.Properties.PreviousKeys) > 0 {
		for _, prevKey := range diskEncryptionSet.Properties.PreviousKeys {
			if prevKey == nil {
				continue
			}

			// Link to Key Vault Vault from PreviousKeys[].SourceVault.ID
			if prevKey.SourceVault != nil && prevKey.SourceVault.ID != nil && *prevKey.SourceVault.ID != "" {
				vaultID := *prevKey.SourceVault.ID
				vaultName := azureshared.ExtractResourceName(vaultID)
				if vaultName != "" {
					// Deduplicate by (type, method, query). QueryTests uses type+query uniqueness.
					if !hasLinkedQuery(azureshared.KeyVaultVault.String(), sdp.QueryMethod_GET, vaultName) {
						extractedScope := azureshared.ExtractScopeFromResourceID(vaultID)
						if extractedScope == "" {
							extractedScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.KeyVaultVault.String(),
								Method: sdp.QueryMethod_GET,
								Query:  vaultName,
								Scope:  extractedScope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Key Vault is deleted/locked down → DES can't access key material (In: true)
								Out: false, // If DES is deleted/modified → Key Vault remains (Out: false)
							},
						})
					}
				}
			}

			// Link to Key Vault Key + DNS from PreviousKeys[].KeyURL (mirrors ActiveKey.KeyURL behavior)
			if prevKey.KeyURL != nil && *prevKey.KeyURL != "" {
				prevKeyURL := *prevKey.KeyURL

				vaultName := azureshared.ExtractVaultNameFromURI(prevKeyURL)
				keyName := azureshared.ExtractKeyNameFromURI(prevKeyURL)
				if vaultName != "" && keyName != "" {
					keyQuery := shared.CompositeLookupKey(vaultName, keyName)
					if !hasLinkedQuery(azureshared.KeyVaultKey.String(), sdp.QueryMethod_GET, keyQuery) {
						// Key Vault URI doesn't contain resource group, use DES scope as best effort
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.KeyVaultKey.String(),
								Method: sdp.QueryMethod_GET,
								Query:  keyQuery,
								Scope:  scope, // Limitation: Key Vault URI doesn't contain resource group info
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Key Vault Key is deleted/modified → DES can't access key material (In: true)
								Out: false, // If DES is deleted/modified → Key remains (Out: false)
							},
						})
					}
				}

				dnsName := azureshared.ExtractDNSFromURL(prevKeyURL)
				if dnsName != "" && !hasLinkedQuery("dns", sdp.QueryMethod_SEARCH, dnsName) {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  dnsName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS names are always linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	// Link to Key Vault Key from Properties.ActiveKey.KeyURL
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keys/get-key/get-key?view=rest-keyvault-keys-2016-10-01
	if diskEncryptionSet.Properties != nil &&
		diskEncryptionSet.Properties.ActiveKey != nil &&
		diskEncryptionSet.Properties.ActiveKey.KeyURL != nil &&
		*diskEncryptionSet.Properties.ActiveKey.KeyURL != "" {
		keyURL := *diskEncryptionSet.Properties.ActiveKey.KeyURL
		vaultName := azureshared.ExtractVaultNameFromURI(keyURL)
		keyName := azureshared.ExtractKeyNameFromURI(keyURL)
		if vaultName != "" && keyName != "" {
			keyQuery := shared.CompositeLookupKey(vaultName, keyName)
			// Key Vault URI doesn't contain resource group, use DES scope as best effort
			if !hasLinkedQuery(azureshared.KeyVaultKey.String(), sdp.QueryMethod_GET, keyQuery) {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  keyQuery,
						Scope:  scope, // Limitation: Key Vault URI doesn't contain resource group info
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Key Vault Key is deleted/modified → DES can't access key material (In: true)
						Out: false, // If DES is deleted/modified → Key remains (Out: false)
					},
				})
			}
		}

		// Link to DNS name (standard library) from KeyURL
		dnsName := azureshared.ExtractDNSFromURL(keyURL)
		if dnsName != "" && !hasLinkedQuery("dns", sdp.QueryMethod_SEARCH, dnsName) {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS names are always linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	// Link to user-assigned managed identities from Identity.UserAssignedIdentities map keys (resource IDs)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30
	if diskEncryptionSet.Identity != nil && diskEncryptionSet.Identity.UserAssignedIdentities != nil {
		for identityID := range diskEncryptionSet.Identity.UserAssignedIdentities {
			if identityID == "" {
				continue
			}
			identityName := azureshared.ExtractResourceName(identityID)
			if identityName == "" {
				continue
			}
			extractedScope := azureshared.ExtractScopeFromResourceID(identityID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  identityName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If identity is deleted/permissions change → DES can't authenticate to Key Vault (In: true)
					Out: false, // If DES is deleted/modified → identity remains (Out: false)
				},
			})
		}
	}

	return sdpItem, nil
}

func (c computeDiskEncryptionSetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskEncryptionSetLookupByName,
	}
}

func (c computeDiskEncryptionSetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeDisk,
		azureshared.KeyVaultVault,
		azureshared.KeyVaultKey,
		azureshared.ManagedIdentityUserAssignedIdentity,
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/disk_encryption_set
func (c computeDiskEncryptionSetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_disk_encryption_set.name",
		},
	}
}
