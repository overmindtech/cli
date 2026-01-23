package manual

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var CloudKMSCryptoKeyVersionLookupByVersion = shared.NewItemTypeLookup("version", gcpshared.CloudKMSCryptoKeyVersion)

// cloudKMSCryptoKeyVersionWrapper wraps the KMS CryptoKeyVersion client for SDP adaptation.
type cloudKMSCryptoKeyVersionWrapper struct {
	client gcpshared.CloudKMSCryptoKeyVersionClient

	*gcpshared.ProjectBase
}

// NewCloudKMSCryptoKeyVersion creates a new cloudKMSCryptoKeyVersionWrapper.
func NewCloudKMSCryptoKeyVersion(client gcpshared.CloudKMSCryptoKeyVersionClient, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &cloudKMSCryptoKeyVersionWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CloudKMSCryptoKeyVersion,
		),
	}
}

func (c cloudKMSCryptoKeyVersionWrapper) IAMPermissions() []string {
	return []string{
		"cloudkms.cryptoKeyVersions.get",
		"cloudkms.cryptoKeyVersions.list",
	}
}

func (c cloudKMSCryptoKeyVersionWrapper) PredefinedRole() string {
	return "roles/cloudkms.viewer"
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
			TerraformMethod:   sdp.QueryMethod_GET,
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

// Get retrieves a KMS CryptoKeyVersion by its name.
// The name must be in the format: projects/{PROJECT_ID}/locations/{LOCATION}/keyRings/{KEY_RING}/cryptoKeys/{CRYPTO_KEY}/cryptoKeyVersions/{VERSION}
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions/get
func (c cloudKMSCryptoKeyVersionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	location := queryParts[0]
	keyRing := queryParts[1]
	cryptoKey := queryParts[2]
	version := queryParts[3]

	name := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s",
		loc.ProjectID, location, keyRing, cryptoKey, version,
	)

	req := &kmspb.GetCryptoKeyVersionRequest{
		Name: name,
	}

	cryptoKeyVersion, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpCryptoKeyVersionToSDPItem(cryptoKeyVersion, loc)
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

// Search searches KMS CryptoKeyVersions for a given CryptoKey and converts them to sdp.Items.
// GET https://cloudkms.googleapis.com/v1/{parent=projects/*/locations/*/keyRings/*/cryptoKeys/*}/cryptoKeyVersions
func (c cloudKMSCryptoKeyVersionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	location := queryParts[0]
	keyRing := queryParts[1]
	cryptoKey := queryParts[2]

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		loc.ProjectID, location, keyRing, cryptoKey,
	)

	it := c.client.List(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: parent,
	})

	var items []*sdp.Item
	for {
		cryptoKeyVersion, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpCryptoKeyVersionToSDPItem(cryptoKeyVersion, loc)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c cloudKMSCryptoKeyVersionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	location := queryParts[0]
	keyRing := queryParts[1]
	cryptoKey := queryParts[2]

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		loc.ProjectID, location, keyRing, cryptoKey,
	)

	it := c.client.List(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: parent,
	})

	for {
		cryptoKeyVersion, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpCryptoKeyVersionToSDPItem(cryptoKeyVersion, loc)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// gcpCryptoKeyVersionToSDPItem converts a GCP CryptoKeyVersion to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
func (c cloudKMSCryptoKeyVersionWrapper) gcpCryptoKeyVersionToSDPItem(cryptoKeyVersion *kmspb.CryptoKeyVersion, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(cryptoKeyVersion)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	// The unique attribute must be the same as the query parameter for the Get method.
	// Which is in the format: location|keyRing|cryptoKey|version
	// We will extract the path parameters from the CryptoKeyVersion name to create a unique lookup key.
	//
	// Example CryptoKeyVersion name: projects/{PROJECT_ID}/locations/{LOCATION}/keyRings/{KEY_RING}/cryptoKeys/{CRYPTO_KEY}/cryptoKeyVersions/{VERSION}
	values := gcpshared.ExtractPathParams(cryptoKeyVersion.GetName(), "locations", "keyRings", "cryptoKeys", "cryptoKeyVersions")
	if len(values) != 4 || values[0] == "" || values[1] == "" || values[2] == "" || values[3] == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid CryptoKeyVersion name: %s", cryptoKeyVersion.GetName()),
		}
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(values...))
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to set unique attribute: %v", err),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           location.ToScope(),
	}

	// Link to parent CryptoKey
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys/get
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.CloudKMSCryptoKey.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(values[0], values[1], values[2]), // location, keyRing, cryptoKey
			Scope:  location.ProjectID,
		},
		// Deleting the parent CryptoKey deletes all CryptoKeyVersions
		// Deleting a CryptoKeyVersion doesn't affect the parent CryptoKey
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	// Link to ImportJob if the key material was imported
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/importJobs/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.importJobs/get
	if importJob := cryptoKeyVersion.GetImportJob(); importJob != "" {
		importJobVals := gcpshared.ExtractPathParams(importJob, "locations", "keyRings", "importJobs")
		if len(importJobVals) == 3 && importJobVals[0] != "" && importJobVals[1] != "" && importJobVals[2] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.CloudKMSImportJob.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(importJobVals...),
					Scope:  location.ProjectID,
				},
				// Deleting the ImportJob doesn't affect the CryptoKeyVersion once imported
				// The CryptoKeyVersion doesn't own the ImportJob
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to EKMConnection if using external key management
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/ekmConnections/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.ekmConnections/get
	if protectionLevel := cryptoKeyVersion.GetProtectionLevel(); protectionLevel == kmspb.ProtectionLevel_EXTERNAL_VPC {
		if externalProtection := cryptoKeyVersion.GetExternalProtectionLevelOptions(); externalProtection != nil {
			if ekmPath := externalProtection.GetEkmConnectionKeyPath(); ekmPath != "" {
				// Extract EKM connection name from the key path
				// EkmConnectionKeyPath format may vary, need to extract connection name carefully
				// For now, we'll attempt to parse it if it follows a standard pattern
				ekmVals := gcpshared.ExtractPathParams(ekmPath, "locations", "ekmConnections")
				if len(ekmVals) == 2 && ekmVals[0] != "" && ekmVals[1] != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.CloudKMSEKMConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(ekmVals...),
							Scope:  location.ProjectID,
						},
						// Deleting the EKM connection makes the CryptoKeyVersion non-functional
						// The CryptoKeyVersion doesn't own the EKM connection
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	// Set health based on CryptoKeyVersion state
	switch cryptoKeyVersion.GetState() {
	case kmspb.CryptoKeyVersion_CRYPTO_KEY_VERSION_STATE_UNSPECIFIED:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case kmspb.CryptoKeyVersion_PENDING_GENERATION, kmspb.CryptoKeyVersion_PENDING_IMPORT:
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case kmspb.CryptoKeyVersion_ENABLED:
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case kmspb.CryptoKeyVersion_DISABLED:
		sdpItem.Health = sdp.Health_HEALTH_WARNING.Enum()
	case kmspb.CryptoKeyVersion_DESTROYED, kmspb.CryptoKeyVersion_DESTROY_SCHEDULED:
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case kmspb.CryptoKeyVersion_IMPORT_FAILED, kmspb.CryptoKeyVersion_GENERATION_FAILED, kmspb.CryptoKeyVersion_EXTERNAL_DESTRUCTION_FAILED:
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case kmspb.CryptoKeyVersion_PENDING_EXTERNAL_DESTRUCTION:
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	default:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	return sdpItem, nil
}
