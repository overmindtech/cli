package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var BatchBatchApplicationLookupByName = shared.NewItemTypeLookup("name", azureshared.BatchBatchApplication)

type batchBatchApplicationWrapper struct {
	client clients.BatchApplicationsClient
	*azureshared.MultiResourceGroupBase
}

// NewBatchBatchApplication returns a SearchableWrapper for Azure Batch applications (child of Batch account).
func NewBatchBatchApplication(client clients.BatchApplicationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &batchBatchApplicationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.BatchBatchApplication,
		),
	}
}

func (b batchBatchApplicationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: accountName and applicationName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]
	applicationName := queryParts[1]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	resp, err := b.client.Get(ctx, rgScope.ResourceGroup, accountName, applicationName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	return b.azureApplicationToSDPItem(&resp.Application, accountName, applicationName, scope)
}

func (b batchBatchApplicationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BatchAccountLookupByName,
		BatchBatchApplicationLookupByName,
	}
}

func (b batchBatchApplicationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: accountName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	pager := b.client.List(ctx, rgScope.ResourceGroup, accountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, b.Type())
		}

		for _, app := range page.Value {
			if app == nil || app.Name == nil {
				continue
			}
			item, sdpErr := b.azureApplicationToSDPItem(app, accountName, *app.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (b batchBatchApplicationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: accountName"), scope, b.Type()))
		return
	}
	accountName := queryParts[0]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, b.Type()))
		return
	}
	pager := b.client.List(ctx, rgScope.ResourceGroup, accountName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, b.Type()))
			return
		}
		for _, app := range page.Value {
			if app == nil || app.Name == nil {
				continue
			}
			item, sdpErr := b.azureApplicationToSDPItem(app, accountName, *app.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (b batchBatchApplicationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BatchAccountLookupByName,
		},
	}
}

func (b batchBatchApplicationWrapper) azureApplicationToSDPItem(app *armbatch.Application, accountName, applicationName, scope string) (*sdp.Item, *sdp.QueryError) {
	if app.Name == nil {
		return nil, azureshared.QueryError(errors.New("application name is nil"), scope, b.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(app, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	if err := attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, applicationName)); err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.BatchBatchApplication.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(app.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to parent Batch Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  accountName,
			Scope:  scope,
		},
	})

	// Link to Application Packages (child resource under this application)
	// Packages are listed under /batchAccounts/{account}/applications/{app}/versions
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchApplicationPackage.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(accountName, applicationName),
			Scope:  scope,
		},
	})

	// Link to default version application package when set (GET to specific child resource)
	if app.Properties != nil && app.Properties.DefaultVersion != nil && *app.Properties.DefaultVersion != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.BatchBatchApplicationPackage.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(accountName, applicationName, *app.Properties.DefaultVersion),
				Scope:  scope,
			},
		})
	}

	return sdpItem, nil
}

func (b batchBatchApplicationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.BatchBatchAccount:               true,
		azureshared.BatchBatchApplicationPackage:    true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/batch_application
func (b batchBatchApplicationWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_batch_application.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute
func (b batchBatchApplicationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Batch/batchAccounts/applications/read",
	}
}

func (b batchBatchApplicationWrapper) PredefinedRole() string {
	return "Azure Batch Account Reader"
}
