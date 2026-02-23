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
)

var StorageFileShareLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageFileShare)

type storageFileShareWrapper struct {
	client clients.FileSharesClient

	*azureshared.MultiResourceGroupBase
}

func NewStorageFileShare(client clients.FileSharesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &storageFileShareWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageFileShare,
		),
	}
}

func (s storageFileShareWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and shareName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	shareName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, storageAccountName, shareName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = s.azureFileShareToSDPItem(&resp.FileShare, storageAccountName, shareName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (s storageFileShareWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StorageFileShareLookupByName,
	}
}

func (s storageFileShareWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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

		for _, fileShare := range page.Value {
			if fileShare.Name == nil {
				continue
			}

			item, sdpErr := s.azureFileShareToSDPItem(&armstorage.FileShare{
				ID:                  fileShare.ID,
				Name:                fileShare.Name,
				Type:                fileShare.Type,
				FileShareProperties: fileShare.Properties,
				Etag:                fileShare.Etag,
			}, storageAccountName, *fileShare.Name, scope,
			)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageFileShareWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
		for _, fileShare := range page.Value {
			if fileShare.Name == nil {
				continue
			}
			item, sdpErr := s.azureFileShareToSDPItem(&armstorage.FileShare{
				ID:                  fileShare.ID,
				Name:                fileShare.Name,
				Type:                fileShare.Type,
				FileShareProperties: fileShare.Properties,
				Etag:                fileShare.Etag,
			}, storageAccountName, *fileShare.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s storageFileShareWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName,
		},
	}
}

func (s storageFileShareWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.StorageAccount: true,
	}
}

func (s storageFileShareWrapper) azureFileShareToSDPItem(fileShare *armstorage.FileShare, storageAccountName, shareName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(fileShare)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(storageAccountName, shareName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageFileShare.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  storageAccountName,
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (s storageFileShareWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share
			// Terraform uses: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{account}/fileServices/default/shares/{share}
			TerraformQueryMap: "azurerm_storage_share.id",
		},
	}
}

func (s storageFileShareWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/fileServices/shares/read",
	}
}

func (s storageFileShareWrapper) PredefinedRole() string {
	return "Storage File Data Privileged Reader"
}
