package manual

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var StorageQueueLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageQueue)

type storageQueuesWrapper struct {
	client clients.QueuesClient

	*azureshared.MultiResourceGroupBase
}

func NewStorageQueues(client clients.QueuesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &storageQueuesWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageQueue,
		),
	}
}

func (s storageQueuesWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: storageAccountName and queueName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	storageAccountName := queryParts[0]
	queueName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, storageAccountName, queueName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureQueueToSDPItem(&resp.Queue, storageAccountName, queueName, scope)
}

func (s storageQueuesWrapper) azureQueueToSDPItem(queue *armstorage.Queue, storageAccountName, queueName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(queue)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(storageAccountName, queueName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageQueue.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Queue is a child of the storage account; queue is affected if account changes, account is not affected by queue changes.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  storageAccountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // Queue depends on storage account; account deletion/change affects the queue.
			Out: false, // Storage account is not affected by queue create/update/delete.
		},
	})

	return sdpItem, nil
}

func (s storageQueuesWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_queue
			// Terraform uses: /subscriptions/{{subscription}}/resourceGroups/{{resourceGroup}}/providers/Microsoft.Storage/storageAccounts/{{storageAccountName}}/queueServices/default/queues/{{queueName}}
			TerraformQueryMap: "azurerm_storage_queue.id",
		},
	}
}

func (s storageQueuesWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
		StorageQueueLookupByName,
	}
}

func (s storageQueuesWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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

		for _, queue := range page.Value {
			if queue.Name == nil || queue.QueueProperties == nil {
				continue
			}

			item, sdpErr := s.azureQueueToSDPItem(&armstorage.Queue{
				ID:   queue.ID,
				Name: queue.Name,
				Type: queue.Type,
				QueueProperties: &armstorage.QueueProperties{
					Metadata: queue.QueueProperties.Metadata,
				},
			}, storageAccountName, *queue.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageQueuesWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			StorageAccountLookupByName,
		},
	}
}

func (s storageQueuesWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.StorageAccount: true,
	}
}

func (s storageQueuesWrapper) IAMPermissions() []string {
	return []string{
		// reference: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/storage#storage-queue-data-reader
		"Microsoft.Storage/storageAccounts/queueServices/queues/read",
	}
}

func (s storageQueuesWrapper) PredefinedRole() string {
	// reference: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/storage#storage-queue-data-reader
	return "Storage Queue Data Reader"
}
