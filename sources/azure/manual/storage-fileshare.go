package manual

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	StorageFileShareLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageFileShare)
)

type storageFileShareWrapper struct {
	client clients.FileSharesClient

	*azureshared.ResourceGroupBase
}

func NewStorageFileShare(client clients.FileSharesClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &storageFileShareWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageFileShare,
		),
	}
}

func (s storageFileShareWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and shareName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	shareName := queryParts[1]

	resp, err := s.client.Get(ctx, s.ResourceGroup(), storageAccountName, shareName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = s.azureFileShareToSDPItem(&resp.FileShare, storageAccountName, shareName)
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

func (s storageFileShareWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: storageAccountName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]

	pager := s.client.List(ctx, s.ResourceGroup(), storageAccountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
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
			}, storageAccountName, *fileShare.Name,
			)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
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

func (s storageFileShareWrapper) azureFileShareToSDPItem(fileShare *armstorage.FileShare, storageAccountName, shareName string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(fileShare)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(storageAccountName, shareName))
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageFileShare.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           s.DefaultScope(),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  storageAccountName,
			Scope:  s.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	return sdpItem, nil
}

func (s storageFileShareWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_share
			TerraformQueryMap: "azurerm_storage_share.name",
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
