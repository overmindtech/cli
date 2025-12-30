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
	StorageTableLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageTable)
)

type storageTablesWrapper struct {
	client clients.TablesClient

	*azureshared.ResourceGroupBase
}

func NewStorageTable(client clients.TablesClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &storageTablesWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageTable,
		),
	}
}

func (s storageTablesWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and tableName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	tableName := queryParts[1]

	resp, err := s.client.Get(ctx, s.ResourceGroup(), storageAccountName, tableName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	item, sdpErr := s.azureTableToSDPItem(&resp.Table, storageAccountName, tableName)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (s storageTablesWrapper) azureTableToSDPItem(table *armstorage.Table, storageAccountName, tableName string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(table)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	err = attributes.Set("id", shared.CompositeLookupKey(storageAccountName, tableName))
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageTable.String(),
		UniqueAttribute: "id",
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
			In:  true,  // Tables ARE affected if storage account changes/deletes
			Out: false, // Tables changes/deletes don't affect storage account
		},
	})

	return sdpItem, nil
}

func (s storageTablesWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_table
			// Terraform uses: /subscriptions/{{sub}}/resourceGroups/{{rg}}/providers/Microsoft.Storage/storageAccounts/{{account}}/tableServices/default/tables/{{table}}
			TerraformQueryMap: "azurerm_storage_table.id",
		},
	}
}

func (s storageTablesWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StorageTableLookupByName,
	}
}

func (s storageTablesWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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

		for _, table := range page.Value {
			if table.Name == nil {
				continue
			}

			item, sdpErr := s.azureTableToSDPItem(&armstorage.Table{
				ID:              table.ID,
				Name:            table.Name,
				Type:            table.Type,
				TableProperties: table.TableProperties,
			}, storageAccountName, *table.Name,
			)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageTablesWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName,
		},
	}
}

func (s storageTablesWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.StorageAccount: true,
	}
}

func (s storageTablesWrapper) IAMPermissions() []string {
	return []string{
		// https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/storage#storage-table-data-reader
		"Microsoft.Storage/storageAccounts/tableServices/tables/read",
	}
}

func (s storageTablesWrapper) PredefinedRole() string {
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/storage#storage-table-data-reader
	return "Storage Table Data Reader"
}
