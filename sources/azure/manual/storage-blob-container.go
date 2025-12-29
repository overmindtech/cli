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
	StorageBlobContainerLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageBlobContainer)
)

type storageBlobContainerWrapper struct {
	client clients.BlobContainersClient

	*azureshared.ResourceGroupBase
}

func NewStorageBlobContainer(client clients.BlobContainersClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &storageBlobContainerWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageBlobContainer,
		),
	}
}

func (s storageBlobContainerWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and containerName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	containerName := queryParts[1]

	resp, err := s.client.Get(ctx, s.ResourceGroup(), storageAccountName, containerName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = s.azureBlobContainerToSDPItem(&resp.BlobContainer, storageAccountName, containerName)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (s storageBlobContainerWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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

		for _, container := range page.Value {
			if container.Name == nil {
				continue
			}

			item, sdpErr := s.azureBlobContainerToSDPItem(&armstorage.BlobContainer{
				ID:                  container.ID,
				Name:                container.Name,
				Type:                container.Type,
				ContainerProperties: container.Properties,
				Etag:                container.Etag,
			}, storageAccountName, *container.Name)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageBlobContainerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StorageBlobContainerLookupByName,
	}
}

func (s storageBlobContainerWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName, // Search by storage account name
		},
	}
}

// PotentialLinks returns the potential links for the blob container wrapper
func (s storageBlobContainerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.StorageAccount,
	)
}

func (s storageBlobContainerWrapper) azureBlobContainerToSDPItem(container *armstorage.BlobContainer, storageAccountName, containerName string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(container)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	err = attributes.Set("id", shared.CompositeLookupKey(storageAccountName, containerName))
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageBlobContainer.String(),
		UniqueAttribute: "name",
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

func (s storageBlobContainerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_container
			TerraformQueryMap: "azurerm_storage_container.name",
		},
	}
}

func (s storageBlobContainerWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/blobServices/containers/read",
	}
}

func (s storageBlobContainerWrapper) PredefinedRole() string {
	return "Storage Blob Data Reader"
}
