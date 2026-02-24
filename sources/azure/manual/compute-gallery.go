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

var ComputeGalleryLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeGallery)

type computeGalleryWrapper struct {
	client clients.GalleriesClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeGallery(client clients.GalleriesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeGalleryWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeGallery,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/galleries/list-by-resource-group
func (c computeGalleryWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, gallery := range page.Value {
			if gallery == nil || gallery.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryToSDPItem(gallery, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeGalleryWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, gallery := range page.Value {
			if gallery == nil || gallery.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryToSDPItem(gallery, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/galleries/get
func (c computeGalleryWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
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
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, galleryName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureGalleryToSDPItem(&resp.Gallery, scope)
}

func (c computeGalleryWrapper) azureGalleryToSDPItem(gallery *armcompute.Gallery, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(gallery, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if gallery.Name == nil {
		return nil, azureshared.QueryError(errors.New("gallery name is nil"), scope, c.Type())
	}
	galleryName := *gallery.Name

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Child resources: list gallery images under this gallery (Search by gallery name)
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGalleryImage.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  galleryName,
			Scope:  scope,
		},
	})

	// Child resources: list gallery applications under this gallery (Search by gallery name)
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGalleryApplication.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  galleryName,
			Scope:  scope,
		},
	})

	// URI-based links from community gallery info: PublisherURI, Eula
	linkedDNSHostnames := make(map[string]struct{})
	seenIPs := make(map[string]struct{})
	if gallery.Properties != nil && gallery.Properties.SharingProfile != nil && gallery.Properties.SharingProfile.CommunityGalleryInfo != nil {
		info := gallery.Properties.SharingProfile.CommunityGalleryInfo
		if info.PublisherURI != nil {
			AppendURILinks(&linkedItemQueries, *info.PublisherURI, linkedDNSHostnames, seenIPs)
		}
		if info.Eula != nil {
			AppendURILinks(&linkedItemQueries, *info.Eula, linkedDNSHostnames, seenIPs)
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeGallery.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(gallery.Tags),
		LinkedItemQueries: linkedItemQueries,
	}

	// Health status from ProvisioningState
	if gallery.Properties != nil && gallery.Properties.ProvisioningState != nil {
		switch *gallery.Properties.ProvisioningState {
		case armcompute.GalleryProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armcompute.GalleryProvisioningStateCreating, armcompute.GalleryProvisioningStateUpdating, armcompute.GalleryProvisioningStateDeleting, armcompute.GalleryProvisioningStateMigrating:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armcompute.GalleryProvisioningStateFailed:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return sdpItem, nil
}

func (c computeGalleryWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeGalleryLookupByName,
	}
}

func (c computeGalleryWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeGalleryImage,
		azureshared.ComputeGalleryApplication,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/shared_image_gallery
func (c computeGalleryWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_shared_image_gallery.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeGalleryWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/galleries/read",
	}
}

func (c computeGalleryWrapper) PredefinedRole() string {
	return "Reader"
}
