package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var (
	ComputeSharedGalleryImageLookupByLocation         = shared.NewItemTypeLookup("location", azureshared.ComputeSharedGalleryImage)
	ComputeSharedGalleryImageLookupByGalleryUniqueName = shared.NewItemTypeLookup("galleryUniqueName", azureshared.ComputeSharedGalleryImage)
	ComputeSharedGalleryImageLookupByName              = shared.NewItemTypeLookup("name", azureshared.ComputeSharedGalleryImage)
)

type computeSharedGalleryImageWrapper struct {
	client clients.SharedGalleryImagesClient
	*azureshared.SubscriptionBase
}

func NewComputeSharedGalleryImage(client clients.SharedGalleryImagesClient, subscriptionID string) sources.SearchableWrapper {
	return &computeSharedGalleryImageWrapper{
		client: client,
		SubscriptionBase: azureshared.NewSubscriptionBase(
			subscriptionID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeSharedGalleryImage,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/shared-gallery-images/get
func (c computeSharedGalleryImageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 3 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 3: location, gallery unique name, and image name"), scope, c.Type())
	}
	location := queryParts[0]
	if location == "" {
		return nil, azureshared.QueryError(errors.New("location cannot be empty"), scope, c.Type())
	}
	galleryUniqueName := queryParts[1]
	if galleryUniqueName == "" {
		return nil, azureshared.QueryError(errors.New("gallery unique name cannot be empty"), scope, c.Type())
	}
	galleryImageName := queryParts[2]
	if galleryImageName == "" {
		return nil, azureshared.QueryError(errors.New("gallery image name cannot be empty"), scope, c.Type())
	}

	resp, err := c.client.Get(ctx, location, galleryUniqueName, galleryImageName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureSharedGalleryImageToSDPItem(&resp.SharedGalleryImage, location, galleryUniqueName, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/shared-gallery-images/list
func (c computeSharedGalleryImageWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2: location and gallery unique name"), scope, c.Type())
	}
	location := queryParts[0]
	if location == "" {
		return nil, azureshared.QueryError(errors.New("location cannot be empty"), scope, c.Type())
	}
	galleryUniqueName := queryParts[1]
	if galleryUniqueName == "" {
		return nil, azureshared.QueryError(errors.New("gallery unique name cannot be empty"), scope, c.Type())
	}

	pager := c.client.NewListPager(location, galleryUniqueName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, image := range page.Value {
			if image == nil || image.Name == nil {
				continue
			}
			item, sdpErr := c.azureSharedGalleryImageToSDPItem(image, location, galleryUniqueName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeSharedGalleryImageWrapper) azureSharedGalleryImageToSDPItem(
	image *armcompute.SharedGalleryImage,
	location,
	galleryUniqueName,
	scope string,
) (*sdp.Item, *sdp.QueryError) {
	if image.Name == nil {
		return nil, azureshared.QueryError(errors.New("shared gallery image name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(image)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	imageName := *image.Name
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(location, galleryUniqueName, imageName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent Shared Gallery: image definition depends on shared gallery (Microsoft.Compute/locations/sharedGalleries)
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeSharedGallery.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(location, galleryUniqueName),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,  // If shared gallery is removed → image is no longer visible
			Out: false, // If image definition is deleted → shared gallery remains
		},
	})

	// URI-based links. Note: armcompute.SharedGalleryImageProperties has no ReleaseNoteURI field (unlike GalleryImage).
	linkedDNSHostnames := make(map[string]struct{})
	seenIPs := make(map[string]struct{})
	if image.Properties != nil {
		if image.Properties.Eula != nil {
			AppendURILinks(&linkedItemQueries, *image.Properties.Eula, linkedDNSHostnames, seenIPs, true, false)
		}
		if image.Properties.PrivacyStatementURI != nil {
			AppendURILinks(&linkedItemQueries, *image.Properties.PrivacyStatementURI, linkedDNSHostnames, seenIPs, true, false)
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeSharedGalleryImage.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		LinkedItemQueries: linkedItemQueries,
	}
	return sdpItem, nil
}

func (c computeSharedGalleryImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSharedGalleryImageLookupByLocation,
		ComputeSharedGalleryImageLookupByGalleryUniqueName,
		ComputeSharedGalleryImageLookupByName,
	}
}

func (c computeSharedGalleryImageWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeSharedGalleryImageLookupByLocation,
			ComputeSharedGalleryImageLookupByGalleryUniqueName,
		},
	}
}

func (c computeSharedGalleryImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeSharedGallery,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// Shared gallery images are read-only views with no direct Terraform resource mapping.
func (c computeSharedGalleryImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return nil
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeSharedGalleryImageWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/locations/sharedGalleries/images/read",
	}
}

func (c computeSharedGalleryImageWrapper) PredefinedRole() string {
	return "Reader"
}
