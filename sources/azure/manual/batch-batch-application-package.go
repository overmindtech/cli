package manual

import (
	"context"
	"errors"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var BatchBatchApplicationPackageLookupByName = shared.NewItemTypeLookup("name", azureshared.BatchBatchApplicationPackage)

type batchBatchApplicationPackageWrapper struct {
	client clients.BatchApplicationPackagesClient
	*azureshared.MultiResourceGroupBase
}

// NewBatchBatchApplicationPackage returns a SearchableWrapper for Azure Batch application packages
// (child of Batch application, grandchild of Batch account).
func NewBatchBatchApplicationPackage(client clients.BatchApplicationPackagesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &batchBatchApplicationPackageWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.BatchBatchApplicationPackage,
		),
	}
}

func (c batchBatchApplicationPackageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 3 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 3 query parts: accountName, applicationName, and versionName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	accountName := queryParts[0]
	applicationName := queryParts[1]
	versionName := queryParts[2]

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, accountName, applicationName, versionName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureApplicationPackageToSDPItem(&resp.ApplicationPackage, accountName, applicationName, versionName, scope)
}

func (c batchBatchApplicationPackageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BatchAccountLookupByName,
		BatchBatchApplicationLookupByName,
		BatchBatchApplicationPackageLookupByName,
	}
}

func (c batchBatchApplicationPackageWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 2 query parts: accountName and applicationName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	accountName := queryParts[0]
	applicationName := queryParts[1]

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.List(ctx, rgScope.ResourceGroup, accountName, applicationName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}

		for _, pkg := range page.Value {
			if pkg == nil || pkg.Name == nil {
				continue
			}
			item, sdpErr := c.azureApplicationPackageToSDPItem(pkg, accountName, applicationName, *pkg.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c batchBatchApplicationPackageWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 2 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 2 query parts: accountName and applicationName"), scope, c.Type()))
		return
	}
	accountName := queryParts[0]
	applicationName := queryParts[1]

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.List(ctx, rgScope.ResourceGroup, accountName, applicationName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, pkg := range page.Value {
			if pkg == nil || pkg.Name == nil {
				continue
			}
			item, sdpErr := c.azureApplicationPackageToSDPItem(pkg, accountName, applicationName, *pkg.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c batchBatchApplicationPackageWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BatchAccountLookupByName,
			BatchBatchApplicationLookupByName,
		},
	}
}

func (c batchBatchApplicationPackageWrapper) azureApplicationPackageToSDPItem(pkg *armbatch.ApplicationPackage, accountName, applicationName, versionName, scope string) (*sdp.Item, *sdp.QueryError) {
	if pkg.Name == nil {
		return nil, azureshared.QueryError(errors.New("application package name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(pkg, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if err := attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, applicationName, versionName)); err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.BatchBatchApplicationPackage.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(pkg.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health status from package state
	if pkg.Properties != nil && pkg.Properties.State != nil {
		switch *pkg.Properties.State {
		case armbatch.PackageStateActive:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armbatch.PackageStatePending:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to parent Batch Application
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchApplication.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(accountName, applicationName),
			Scope:  scope,
		},
	})

	// Link to parent Batch Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  accountName,
			Scope:  scope,
		},
	})

	// Link to StorageURL DNS name (Azure Storage blob endpoint hosting the package)
	if pkg.Properties != nil && pkg.Properties.StorageURL != nil && *pkg.Properties.StorageURL != "" {
		u, parseErr := url.Parse(*pkg.Properties.StorageURL)
		if parseErr == nil && u.Hostname() != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  u.Hostname(),
					Scope:  "global",
				},
			})
		}
	}

	return sdpItem, nil
}

func (c batchBatchApplicationPackageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.BatchBatchApplication: true,
		azureshared.BatchBatchAccount:     true,
		stdlib.NetworkDNS:                 true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftbatch
func (c batchBatchApplicationPackageWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Batch/batchAccounts/applications/versions/read",
	}
}

func (c batchBatchApplicationPackageWrapper) PredefinedRole() string {
	return "Azure Batch Account Reader"
}
