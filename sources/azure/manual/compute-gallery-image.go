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

var ComputeGalleryImageLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeGalleryImage)

type computeGalleryImageWrapper struct {
	client clients.GalleryImagesClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeGalleryImage(client clients.GalleryImagesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeGalleryImageWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeGalleryImage,
		),
	}
}

func (c computeGalleryImageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2 and be the gallery name and gallery image name"), scope, c.Type())
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		return nil, azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type())
	}
	galleryImageName := queryParts[1]
	if galleryImageName == "" {
		return nil, azureshared.QueryError(errors.New("gallery image name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, galleryName, galleryImageName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureGalleryImageToSDPItem(&resp.GalleryImage, galleryName, scope)
}

func (c computeGalleryImageWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, galleryImage := range page.Value {
			if galleryImage == nil || galleryImage.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryImageToSDPItem(galleryImage, galleryName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeGalleryImageWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
		for _, galleryImage := range page.Value {
			if galleryImage == nil || galleryImage.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryImageToSDPItem(galleryImage, galleryName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeGalleryImageWrapper) azureGalleryImageToSDPItem(
	galleryImage *armcompute.GalleryImage,
	galleryName,
	scope string,
) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(galleryImage, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if galleryImage.Name == nil {
		return nil, azureshared.QueryError(errors.New("gallery image name is nil"), scope, c.Type())
	}
	galleryImageName := *galleryImage.Name
	if galleryImageName == "" {
		return nil, azureshared.QueryError(errors.New("gallery image name cannot be empty"), scope, c.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(galleryName, galleryImageName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent Gallery: image definition depends on gallery
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGallery.String(),
			Method: sdp.QueryMethod_GET,
			Query:  galleryName,
			Scope:  scope,
		},
	})

	// URI-based links: Eula, PrivacyStatementURI, ReleaseNoteURI
	linkedDNSHostnames := make(map[string]struct{})
	seenIPs := make(map[string]struct{})
	if galleryImage.Properties != nil {
		if galleryImage.Properties.Eula != nil {
			AppendURILinks(&linkedItemQueries, *galleryImage.Properties.Eula, linkedDNSHostnames, seenIPs)
		}
		if galleryImage.Properties.PrivacyStatementURI != nil {
			AppendURILinks(&linkedItemQueries, *galleryImage.Properties.PrivacyStatementURI, linkedDNSHostnames, seenIPs)
		}
		if galleryImage.Properties.ReleaseNoteURI != nil {
			AppendURILinks(&linkedItemQueries, *galleryImage.Properties.ReleaseNoteURI, linkedDNSHostnames, seenIPs)
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeGalleryImage.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(galleryImage.Tags),
		LinkedItemQueries: linkedItemQueries,
	}
	return sdpItem, nil
}

func (c computeGalleryImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeGalleryLookupByName,
		ComputeGalleryImageLookupByName,
	}
}

func (c computeGalleryImageWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeGalleryLookupByName,
		},
	}
}

func (c computeGalleryImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeGallery,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/shared_image
func (c computeGalleryImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// example id: /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Compute/galleries/gallery1/images/image1
			TerraformQueryMap: "azurerm_shared_image.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeGalleryImageWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/galleries/images/read",
	}
}

func (c computeGalleryImageWrapper) PredefinedRole() string {
	return "Reader"
}
