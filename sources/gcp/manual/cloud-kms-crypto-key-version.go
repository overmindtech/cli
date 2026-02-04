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

var CloudKMSCryptoKeyVersionLookupByVersion = shared.NewItemTypeLookup("version", gcpshared.CloudKMSCryptoKeyVersion)

// cloudKMSCryptoKeyVersionWrapper wraps the KMS CryptoKeyVersion operations using CloudKMSAssetLoader.
type cloudKMSCryptoKeyVersionWrapper struct {
	loader *gcpshared.CloudKMSAssetLoader

	*gcpshared.ProjectBase
}

// NewCloudKMSCryptoKeyVersion creates a new cloudKMSCryptoKeyVersionWrapper.
func NewCloudKMSCryptoKeyVersion(loader *gcpshared.CloudKMSAssetLoader, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &cloudKMSCryptoKeyVersionWrapper{
		loader: loader,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CloudKMSCryptoKeyVersion,
		),
	}
}

func (c cloudKMSCryptoKeyVersionWrapper) IAMPermissions() []string {
	return []string{
		"cloudasset.assets.listResource",
	}
}

func (c cloudKMSCryptoKeyVersionWrapper) PredefinedRole() string {
	return "roles/cloudasset.viewer"
}

// PotentialLinks returns the potential links for the CryptoKeyVersion wrapper.
func (c cloudKMSCryptoKeyVersionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKey,
		gcpshared.CloudKMSImportJob,
		gcpshared.CloudKMSEKMConnection,
	)
}

// TerraformMappings returns the Terraform mappings for the CryptoKeyVersion wrapper.
func (c cloudKMSCryptoKeyVersionWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_crypto_key_version
			// ID format: projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}
			// The framework automatically intercepts queries starting with "projects/" and converts
			// them to GET operations by extracting the last N path parameters (based on GetLookups count).
			TerraformQueryMap: "google_kms_crypto_key_version.id",
		},
	}
}

// GetLookups returns the lookups for the CryptoKeyVersion wrapper.
func (c cloudKMSCryptoKeyVersionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		CloudKMSCryptoKeyRingLookupByLocation,
		CloudKMSCryptoKeyRingLookupByName,
		CloudKMSCryptoKeyLookupByName,
		CloudKMSCryptoKeyVersionLookupByVersion,
	}
}

// Get retrieves a KMS CryptoKeyVersion by its unique attribute (location|keyRing|cryptoKey|version).
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSCryptoKeyVersionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
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

// SearchLookups returns the lookups for the CryptoKeyVersion wrapper.
func (c cloudKMSCryptoKeyVersionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
			CloudKMSCryptoKeyRingLookupByName,
			CloudKMSCryptoKeyLookupByName,
		},
	}
}

// Search searches KMS CryptoKeyVersions by cryptoKey (location|keyRing|cryptoKey).
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSCryptoKeyVersionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.SearchStream(ctx, stream, cache, cacheKey, scope, queryParts...)
	})
}

// SearchStream streams CryptoKeyVersions matching the search criteria (location|keyRing|cryptoKey).
func (c cloudKMSCryptoKeyVersionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, _ sdpcache.Cache, _ sdpcache.CacheKey, scope string, queryParts ...string) {
	_, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	// CryptoKeyVersion search is by location|keyRing|cryptoKey
	searchQuery := shared.CompositeLookupKey(queryParts[0], queryParts[1], queryParts[2])
	c.loader.SearchItems(ctx, stream, scope, c.Type(), searchQuery)
}
