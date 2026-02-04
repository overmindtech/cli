package manual

import (
	"context"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	CloudKMSCryptoKeyRingLookupByName     = shared.NewItemTypeLookup("name", gcpshared.CloudKMSKeyRing)
	CloudKMSCryptoKeyRingLookupByLocation = shared.NewItemTypeLookup("location", gcpshared.CloudKMSKeyRing)
)

// cloudKMSKeyRingWrapper wraps the KMS KeyRing operations using CloudKMSAssetLoader.
type cloudKMSKeyRingWrapper struct {
	loader *gcpshared.CloudKMSAssetLoader

	*gcpshared.ProjectBase
}

// NewCloudKMSKeyRing creates a new cloudKMSKeyRingWrapper.
func NewCloudKMSKeyRing(loader *gcpshared.CloudKMSAssetLoader, locations []gcpshared.LocationInfo) sources.SearchableListableWrapper {
	return &cloudKMSKeyRingWrapper{
		loader: loader,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CloudKMSKeyRing,
		),
	}
}

func (c cloudKMSKeyRingWrapper) IAMPermissions() []string {
	return []string{
		"cloudasset.assets.listResource",
	}
}

func (c cloudKMSKeyRingWrapper) PredefinedRole() string {
	return "roles/cloudasset.viewer"
}

// PotentialLinks returns the potential links for the kms key ring
func (c cloudKMSKeyRingWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.IAMPolicy,
		gcpshared.CloudKMSCryptoKey,
	)
}

// TerraformMappings returns the Terraform mappings for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_key_ring
			// ID format: projects/{{project}}/locations/{{location}}/keyRings/{{name}}
			// The framework automatically intercepts queries starting with "projects/" and converts
			// them to GET operations by extracting the last N path parameters (based on GetLookups count).
			TerraformQueryMap: "google_kms_key_ring.id",
		},
	}
}

// GetLookups returns the lookups for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		CloudKMSCryptoKeyRingLookupByLocation,
		CloudKMSCryptoKeyRingLookupByName,
	}
}

// Get retrieves a KMS KeyRing by its unique attribute (location|keyRingName).
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSKeyRingWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	_, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	uniqueAttr := shared.CompositeLookupKey(queryParts...)
	return c.loader.GetItem(ctx, scope, c.Type(), uniqueAttr)
}

// SearchLookups returns the lookups for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
		},
	}
}

// Search searches KMS KeyRings by location.
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSKeyRingWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.SearchStream(ctx, stream, cache, cacheKey, scope, queryParts...)
	})
}

// SearchStream streams KeyRings matching the search criteria (location).
func (c cloudKMSKeyRingWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, _ sdpcache.Cache, _ sdpcache.CacheKey, scope string, queryParts ...string) {
	_, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	// KeyRing search is by location only
	location := queryParts[0]
	c.loader.SearchItems(ctx, stream, scope, c.Type(), location)
}

// List lists all KMS KeyRings in the project.
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSKeyRingWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

// ListStream streams all KeyRings in the project.
func (c cloudKMSKeyRingWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, _ sdpcache.Cache, _ sdpcache.CacheKey, scope string) {
	_, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	c.loader.ListItems(ctx, stream, scope, c.Type())
}
