package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var KeyVaultKeyLookupByName = shared.NewItemTypeLookup("name", azureshared.KeyVaultKey)

type keyvaultKeyWrapper struct {
	client clients.KeysClient

	*azureshared.MultiResourceGroupBase
}

func NewKeyVaultKey(client clients.KeysClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &keyvaultKeyWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.KeyVaultKey,
		),
	}
}

func (k keyvaultKeyWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Get requires 2 query parts: vaultName and keyName"), scope, k.Type())
	}

	vaultName := queryParts[0]
	if vaultName == "" {
		return nil, azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type())
	}

	keyName := queryParts[1]
	if keyName == "" {
		return nil, azureshared.QueryError(errors.New("keyName cannot be empty"), scope, k.Type())
	}

	rgScope, err := k.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}
	resp, err := k.client.Get(ctx, rgScope.ResourceGroup, vaultName, keyName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	return k.azureKeyToSDPItem(&resp.Key, vaultName, keyName, scope)
}

func (k keyvaultKeyWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Search requires 1 query part: vaultName"), scope, k.Type())
	}

	vaultName := queryParts[0]
	if vaultName == "" {
		return nil, azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type())
	}

	rgScope, err := k.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}
	pager := k.client.NewListPager(rgScope.ResourceGroup, vaultName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, k.Type())
		}
		for _, key := range page.Value {
			if key.Name == nil {
				continue
			}
			var keyVaultName string
			if key.ID != nil && *key.ID != "" {
				vaultParams := azureshared.ExtractPathParamsFromResourceID(*key.ID, []string{"vaults"})
				if len(vaultParams) > 0 {
					keyVaultName = vaultParams[0]
				}
			}
			if keyVaultName == "" {
				keyVaultName = vaultName
			}
			item, sdpErr := k.azureKeyToSDPItem(key, keyVaultName, *key.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (k keyvaultKeyWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: vaultName"), scope, k.Type()))
		return
	}
	vaultName := queryParts[0]
	if vaultName == "" {
		stream.SendError(azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type()))
		return
	}

	rgScope, err := k.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, k.Type()))
		return
	}
	pager := k.client.NewListPager(rgScope.ResourceGroup, vaultName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, k.Type()))
			return
		}
		for _, key := range page.Value {
			if key.Name == nil {
				continue
			}
			var keyVaultName string
			if key.ID != nil && *key.ID != "" {
				vaultParams := azureshared.ExtractPathParamsFromResourceID(*key.ID, []string{"vaults"})
				if len(vaultParams) > 0 {
					keyVaultName = vaultParams[0]
				}
			}
			if keyVaultName == "" {
				keyVaultName = vaultName
			}
			item, sdpErr := k.azureKeyToSDPItem(key, keyVaultName, *key.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (k keyvaultKeyWrapper) azureKeyToSDPItem(key *armkeyvault.Key, vaultName, keyName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(key, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	if key.Name == nil {
		return nil, azureshared.QueryError(errors.New("key name is nil"), scope, k.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(vaultName, keyName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.KeyVaultKey.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(key.Tags),
	}

	if key.ID != nil && *key.ID != "" {
		vaultParams := azureshared.ExtractPathParamsFromResourceID(*key.ID, []string{"vaults"})
		if len(vaultParams) > 0 {
			extractedVaultName := vaultParams[0]
			if extractedVaultName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(*key.ID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  extractedVaultName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	var linkedDNSName string
	if key.Properties != nil && key.Properties.KeyURI != nil && *key.Properties.KeyURI != "" {
		keyURI := *key.Properties.KeyURI
		dnsName := azureshared.ExtractDNSFromURL(keyURI)
		if dnsName != "" {
			linkedDNSName = dnsName
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
			})
		}
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  keyURI,
				Scope:  "global",
			},
		})
	}

	if key.Properties != nil && key.Properties.KeyURIWithVersion != nil && *key.Properties.KeyURIWithVersion != "" {
		keyURIWithVersion := *key.Properties.KeyURIWithVersion
		dnsName := azureshared.ExtractDNSFromURL(keyURIWithVersion)
		if dnsName != "" && dnsName != linkedDNSName {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
			})
		}
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  keyURIWithVersion,
				Scope:  "global",
			},
		})
	}

	return sdpItem, nil
}

func (k keyvaultKeyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		KeyVaultVaultLookupByName, // First key: vault name (queryParts[0])
		KeyVaultKeyLookupByName,   // Second key: key name (queryParts[1])
	}
}

func (k keyvaultKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			KeyVaultVaultLookupByName,
		},
	}
}

func (k keyvaultKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_key_vault_key.id",
		},
	}
}

func (k keyvaultKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.KeyVaultVault,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
	)
}

func (k keyvaultKeyWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.KeyVault/vaults/keys/read",
	}
}

func (k keyvaultKeyWrapper) PredefinedRole() string {
	return "Reader"
}
