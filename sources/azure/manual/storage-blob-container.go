package manual

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var StorageBlobContainerLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageBlobContainer)

type storageBlobContainerWrapper struct {
	client clients.BlobContainersClient

	*azureshared.MultiResourceGroupBase
}

func NewStorageBlobContainer(client clients.BlobContainersClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &storageBlobContainerWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageBlobContainer,
		),
	}
}

func (s storageBlobContainerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and containerName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	containerName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, storageAccountName, containerName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = s.azureBlobContainerToSDPItem(&resp.BlobContainer, storageAccountName, containerName, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (s storageBlobContainerWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(fmt.Errorf("queryParts must be 1 query part: storageAccountName, got %d", len(queryParts)), scope, s.Type())
	}
	storageAccountName := queryParts[0]
	if storageAccountName == "" {
		return nil, azureshared.QueryError(fmt.Errorf("storageAccountName cannot be empty"), scope, s.Type())
	}
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
			}, storageAccountName, *container.Name, scope)
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
		azureshared.StorageEncryptionScope,
		stdlib.NetworkHTTP,
		stdlib.NetworkDNS,
	)
}

func (s storageBlobContainerWrapper) azureBlobContainerToSDPItem(container *armstorage.BlobContainer, storageAccountName, containerName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(container)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(storageAccountName, containerName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageBlobContainer.String(),
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
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	// Link to DNS name (standard library) from blob container URI
	// Blob container URI format: https://{storageAccountName}.blob.core.windows.net/{containerName}
	// Any attribute containing a DNS name should create a LinkedItemQuery for dns type
	blobContainerURI := fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAccountName, containerName)
	dnsName := azureshared.ExtractDNSFromURL(blobContainerURI)
	if dnsName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  dnsName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // If DNS name is unavailable → blob container becomes inaccessible (In: true)
				Out: true, // If blob container is deleted → DNS name may still be used by other resources (Out: true)
			}, // Blob container depends on DNS name for endpoint resolution
		})
	}

	// Link to stdlib.NetworkHTTP for blob container URI
	if strings.HasPrefix(blobContainerURI, "https://") {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  blobContainerURI,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // If HTTP endpoint is unavailable → blob container becomes inaccessible (In: true)
				Out: true, // If blob container is deleted → HTTP endpoint may still be used by other resources (Out: true)
			}, // Blob container depends on HTTP endpoint for access
		})
	}

	// Link to Storage Encryption Scope when container uses a default encryption scope
	if container.ContainerProperties != nil && container.ContainerProperties.DefaultEncryptionScope != nil && *container.ContainerProperties.DefaultEncryptionScope != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.StorageEncryptionScope.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(storageAccountName, *container.ContainerProperties.DefaultEncryptionScope),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // If encryption scope is removed or changed → container's default encryption is affected
				Out: false, // Container deletion does not affect the encryption scope
			},
		})
	}

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
