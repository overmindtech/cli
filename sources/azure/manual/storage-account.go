package manual

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	StorageAccountLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageAccount)
)

type storageAccountWrapper struct {
	client clients.StorageAccountsClient

	*azureshared.ResourceGroupBase
}

func NewStorageAccount(client clients.StorageAccountsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &storageAccountWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageAccount,
		),
	}
}

func (s storageAccountWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := s.client.List(s.ResourceGroup())

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
		}

		for _, account := range page.Value {
			if account.Name == nil {
				continue
			}

			item, sdpErr := s.azureStorageAccountToSDPItem(account, *account.Name)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageAccountWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: name",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]
	if accountName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "name cannot be empty",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}

	resp, err := s.client.Get(ctx, s.ResourceGroup(), accountName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	return s.azureStorageAccountToSDPItem(&resp.Account, accountName)
}

func (s storageAccountWrapper) azureStorageAccountToSDPItem(account *armstorage.Account, accountName string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(account)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	err = attributes.Set("id", accountName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageAccount.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           s.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(account.Tags),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageBlobContainer.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  s.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if blob containers change
			Out: true,  // Blob containers ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageFileShare.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  s.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if file shares change
			Out: true,  // File shares ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageTable.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  s.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if tables change
			Out: true,  // Tables ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageQueue.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  s.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if queues change
			Out: true,  // Queues ARE affected if storage account changes/deletes
		},
	})

	return sdpItem, nil
}

func (s storageAccountWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
	}
}

// PotentialLinks returns the potential links for the storage account wrapper
func (s storageAccountWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.StorageBlobContainer,
		azureshared.StorageFileShare,
		azureshared.StorageTable,
		azureshared.StorageQueue,
	)
}

func (s storageAccountWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account
			TerraformQueryMap: "azurerm_storage_account.name",
		},
	}
}

func (s storageAccountWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/read",
	}
}

func (s storageAccountWrapper) PredefinedRole() string {
	return "Reader" //there is no predefined role for storage accounts, so we use the most restrictive role (Reader)
}
