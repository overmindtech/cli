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

var CloudKMSCryptoKeyLookupByName = shared.NewItemTypeLookup("name", gcpshared.CloudKMSCryptoKey)

// cloudKMSCryptoKeyWrapper wraps the KMS CryptoKey operations using CloudKMSAssetLoader.
type cloudKMSCryptoKeyWrapper struct {
	loader *gcpshared.CloudKMSAssetLoader

	*gcpshared.ProjectBase
}

// NewCloudKMSCryptoKey creates a new cloudKMSCryptoKeyWrapper.
func NewCloudKMSCryptoKey(loader *gcpshared.CloudKMSAssetLoader, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &cloudKMSCryptoKeyWrapper{
		loader: loader,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CloudKMSCryptoKey,
		),
	}
}

func (c cloudKMSCryptoKeyWrapper) IAMPermissions() []string {
	return []string{
		"cloudasset.assets.listResource",
	}
}

func (c cloudKMSCryptoKeyWrapper) PredefinedRole() string {
	return "roles/cloudasset.viewer"
}

// PotentialLinks returns the potential links for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.CloudKMSImportJob,
		gcpshared.CloudKMSEKMConnection,
		gcpshared.IAMPolicy,
		gcpshared.CloudKMSKeyRing,
	)
}

// TerraformMappings returns the Terraform mappings for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_crypto_key
			// ID format: projects/{{project}}/locations/{{location}}/keyRings/{{keyRing}}/cryptoKeys/{{name}}
			// The framework automatically intercepts queries starting with "projects/" and converts
			// them to GET operations by extracting the last N path parameters (based on GetLookups count).
			TerraformQueryMap: "google_kms_crypto_key.id",
		},
	}
}

// GetLookups returns the lookups for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		CloudKMSCryptoKeyRingLookupByLocation,
		CloudKMSCryptoKeyRingLookupByName,
		CloudKMSCryptoKeyLookupByName,
	}
}

// Get retrieves a KMS CryptoKey by its unique attribute (location|keyRing|cryptoKeyName).
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSCryptoKeyWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
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

// SearchLookups returns the lookups for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
			CloudKMSCryptoKeyRingLookupByName,
		},
	}
}

// Search searches KMS CryptoKeys by keyRing (location|keyRing).
// Data is loaded via Cloud Asset API and cached in sdpcache.
func (c cloudKMSCryptoKeyWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.SearchStream(ctx, stream, cache, cacheKey, scope, queryParts...)
	})
}

// SearchStream streams CryptoKeys matching the search criteria (location|keyRing).
func (c cloudKMSCryptoKeyWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, _ sdpcache.Cache, _ sdpcache.CacheKey, scope string, queryParts ...string) {
	_, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	// CryptoKey search is by location|keyRing
	searchQuery := shared.CompositeLookupKey(queryParts[0], queryParts[1])
	c.loader.SearchItems(ctx, stream, scope, c.Type(), searchQuery)
}
