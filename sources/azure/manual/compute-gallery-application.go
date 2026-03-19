package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeGalleryApplicationLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeGalleryApplication)

type computeGalleryApplicationWrapper struct {
	client clients.GalleryApplicationsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeGalleryApplication(client clients.GalleryApplicationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeGalleryApplicationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeGalleryApplication,
		),
	}
}

func (c computeGalleryApplicationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2 and be the gallery name and gallery application name"), scope, c.Type())
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		return nil, azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type())
	}
	galleryApplicationName := queryParts[1]
	if galleryApplicationName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, galleryName, galleryApplicationName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureGalleryApplicationToSDPItem(&resp.GalleryApplication, galleryName, scope)
}

func (c computeGalleryApplicationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the gallery name"), scope, c.Type())
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		return nil, azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByGalleryPager(rgScope.ResourceGroup, galleryName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, galleryApplication := range page.Value {
			if galleryApplication == nil || galleryApplication.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryApplicationToSDPItem(galleryApplication, galleryName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeGalleryApplicationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) != 1 {
		stream.SendError(azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the gallery name"), scope, c.Type()))
		return
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		stream.SendError(azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}

	pager := c.client.NewListByGalleryPager(rgScope.ResourceGroup, galleryName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, galleryApplication := range page.Value {
			if galleryApplication == nil || galleryApplication.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryApplicationToSDPItem(galleryApplication, galleryName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeGalleryApplicationWrapper) azureGalleryApplicationToSDPItem(
	galleryApplication *armcompute.GalleryApplication,
	galleryName,
	scope string,
) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(galleryApplication, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if galleryApplication.Name == nil {
		return nil, azureshared.QueryError(errors.New("gallery application name is nil"), scope, c.Type())
	}
	galleryApplicationName := *galleryApplication.Name
	if galleryApplicationName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application name cannot be empty"), scope, c.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(galleryName, galleryApplicationName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent Gallery: application depends on gallery
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGallery.String(),
			Method: sdp.QueryMethod_GET,
			Query:  galleryName,
			Scope:  scope,
		},
	})

	// Child: list gallery application versions under this application (Search by gallery name + application name)
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGalleryApplicationVersion.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(galleryName, galleryApplicationName),
			Scope:  scope,
		},
	})

	// URI-based links: Eula, PrivacyStatementURI, ReleaseNoteURI
	linkedDNSHostnames := make(map[string]struct{})
	seenIPs := make(map[string]struct{})
	if galleryApplication.Properties != nil {
		if galleryApplication.Properties.Eula != nil && *galleryApplication.Properties.Eula != "" {
			AppendURILinks(&linkedItemQueries, *galleryApplication.Properties.Eula, linkedDNSHostnames, seenIPs)
		}
		if galleryApplication.Properties.PrivacyStatementURI != nil && *galleryApplication.Properties.PrivacyStatementURI != "" {
			AppendURILinks(&linkedItemQueries, *galleryApplication.Properties.PrivacyStatementURI, linkedDNSHostnames, seenIPs)
		}
		if galleryApplication.Properties.ReleaseNoteURI != nil && *galleryApplication.Properties.ReleaseNoteURI != "" {
			AppendURILinks(&linkedItemQueries, *galleryApplication.Properties.ReleaseNoteURI, linkedDNSHostnames, seenIPs)
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeGalleryApplication.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(galleryApplication.Tags),
		LinkedItemQueries: linkedItemQueries,
	}
	return sdpItem, nil
}

func (c computeGalleryApplicationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeGalleryLookupByName,
		ComputeGalleryApplicationLookupByName,
	}
}

func (c computeGalleryApplicationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeGalleryLookupByName,
		},
	}
}

func (c computeGalleryApplicationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeGallery,
		azureshared.ComputeGalleryApplicationVersion,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/gallery_application
func (c computeGalleryApplicationWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_gallery_application.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeGalleryApplicationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/galleries/applications/read",
	}
}

func (c computeGalleryApplicationWrapper) PredefinedRole() string {
	return "Reader"
}
