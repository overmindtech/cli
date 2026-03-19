package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var StorageEncryptionScopeLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageEncryptionScope)

type storageEncryptionScopeWrapper struct {
	client clients.EncryptionScopesClient

	*azureshared.MultiResourceGroupBase
}

func NewStorageEncryptionScope(client clients.EncryptionScopesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &storageEncryptionScopeWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageEncryptionScope,
		),
	}
}

func (s storageEncryptionScopeWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and encryptionScopeName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	encryptionScopeName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, storageAccountName, encryptionScopeName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item, sdpErr := s.azureEncryptionScopeToSDPItem(&resp.EncryptionScope, storageAccountName, encryptionScopeName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (s storageEncryptionScopeWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StorageEncryptionScopeLookupByName,
	}
}

func (s storageEncryptionScopeWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: storageAccountName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.List(ctx, rgScope.ResourceGroup, storageAccountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, encScope := range page.Value {
			if encScope.Name == nil {
				continue
			}

			item, sdpErr := s.azureEncryptionScopeToSDPItem(encScope, storageAccountName, *encScope.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageEncryptionScopeWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: storageAccountName"), scope, s.Type()))
		return
	}
	storageAccountName := queryParts[0]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.List(ctx, rgScope.ResourceGroup, storageAccountName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, encScope := range page.Value {
			if encScope.Name == nil {
				continue
			}
			item, sdpErr := s.azureEncryptionScopeToSDPItem(encScope, storageAccountName, *encScope.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s storageEncryptionScopeWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName,
		},
	}
}

func (s storageEncryptionScopeWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.StorageAccount: true,
		azureshared.KeyVaultVault:  true,
		azureshared.KeyVaultKey:    true,
		stdlib.NetworkDNS:          true,
	}
}

func (s storageEncryptionScopeWrapper) azureEncryptionScopeToSDPItem(encScope *armstorage.EncryptionScope, storageAccountName, encryptionScopeName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(encScope, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(storageAccountName, encryptionScopeName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	item := &sdp.Item{
		Type:            azureshared.StorageEncryptionScope.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  storageAccountName,
			Scope:  scope,
		},
	})

	// Link to Key Vault when encryption scope uses customer-managed keys (source Microsoft.KeyVault)
	if encScope.EncryptionScopeProperties != nil && encScope.EncryptionScopeProperties.KeyVaultProperties != nil && encScope.EncryptionScopeProperties.KeyVaultProperties.KeyURI != nil {
		keyURI := *encScope.EncryptionScopeProperties.KeyVaultProperties.KeyURI
		vaultName := azureshared.ExtractVaultNameFromURI(keyURI)
		keyName := azureshared.ExtractKeyNameFromURI(keyURI)
		if vaultName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  scope,
				},
			})
		}
		if vaultName != "" && keyName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(vaultName, keyName),
					Scope:  scope,
				},
			})
		}
		if dnsName := azureshared.ExtractDNSFromURL(keyURI); dnsName != "" {
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

	if encScope.EncryptionScopeProperties != nil && encScope.EncryptionScopeProperties.State != nil {
		switch *encScope.EncryptionScopeProperties.State {
		case armstorage.EncryptionScopeStateEnabled:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case armstorage.EncryptionScopeStateDisabled:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return item, nil
}

func (s storageEncryptionScopeWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_storage_encryption_scope.id",
		},
	}
}

func (s storageEncryptionScopeWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/encryptionScopes/read",
	}
}

func (s storageEncryptionScopeWrapper) PredefinedRole() string {
	return "Reader"
}
