package manual

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	CloudKMSCryptoKey = shared.NewItemType(gcpshared.GCP, gcpshared.CloudKMS, gcpshared.CryptoKey)

	CloudKMSCryptoKeyLookupByName = shared.NewItemTypeLookup("name", CloudKMSCryptoKey)
)

// cloudKMSCryptoKeyWrapper wraps the KMS CryptoKey client for SDP adaptation.
type cloudKMSCryptoKeyWrapper struct {
	client gcpshared.CloudKMSCryptoKeyClient

	*gcpshared.ProjectBase
}

// NewCloudKMSCryptoKey creates a new cloudKMSCryptoKeyWrapper.
func NewCloudKMSCryptoKey(client gcpshared.CloudKMSCryptoKeyClient, projectID string) sources.SearchableWrapper {
	return &cloudKMSCryptoKeyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			CloudKMSCryptoKey,
		),
	}
}

// PotentialLinks returns the potential links for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		CloudKMSCryptoKeyVersion,
		CloudKMSEKMConnection,
		IAMPolicy,
	)
}

// TerraformMappings returns the Terraform mappings for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_kms_crypto_key.name",
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

// Get retrieves a KMS CryptoKey by its name.
// The name must be in the format: projects/{PROJECT_ID}/locations/{LOCATION}/keyRings/{KEY_RING}/cryptoKeys/{CRYPTO_KEY}
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys/get
func (c cloudKMSCryptoKeyWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location := queryParts[0]
	keyRing := queryParts[1]
	cryptoKeyName := queryParts[2]

	name := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		c.ProjectID(), location, keyRing, cryptoKeyName,
	)

	req := &kmspb.GetCryptoKeyRequest{
		Name: name,
	}

	cryptoKey, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpCryptoKeyToSDPItem(cryptoKey)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// GetLookups returns the lookups for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
			CloudKMSCryptoKeyRingLookupByName,
		},
	}
}

// List lists KMS CryptoKeys and converts them to sdp.Items.
// GET https://cloudkms.googleapis.com/v1/{parent=projects/*/locations/*/keyRings/*}/cryptoKeys
func (c cloudKMSCryptoKeyWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location := queryParts[0]
	keyRing := queryParts[1]

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s",
		c.ProjectID(), location, keyRing,
	)

	it := c.client.List(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: parent,
	})

	var items []*sdp.Item
	for {
		cryptoKey, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := c.gcpCryptoKeyToSDPItem(cryptoKey)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpCryptoKeyToSDPItem converts a GCP CryptoKey to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys
func (c cloudKMSCryptoKeyWrapper) gcpCryptoKeyToSDPItem(cryptoKey *kmspb.CryptoKey) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(cryptoKey, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            CloudKMSCryptoKey.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            cryptoKey.GetLabels(),
	}

	// The resource name of the primary CryptoKeyVersion for this CryptoKey.
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions/get
	// Attribute link: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys#:~:text=keyRings/*/cryptoKeys/*.-,primary,-object%20(CryptoKeyVersion
	if primary := cryptoKey.GetPrimary(); primary != nil {
		if name := primary.GetName(); name != "" {
			// Parsing them all together to improve readability
			location := gcpshared.ExtractPathParam("locations", name)
			keyRing := gcpshared.ExtractPathParam("keyRings", name)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", name)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", name)

			// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					//Note: Not exactly sure of the relationships, so settin both as true just in case
					//If all versions of a crypto key are deleted it will become non-functional
					//CryptoKey can only be deleted if all versions are deleted
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				})
			}
		}
	}

	// The EKM connection used by the key material, if applicable.
	// Only applicable if CryptoKeyVersions have a ProtectionLevel of EXTERNAL_VPC.
	// Primary is the CryptoKeyVersion that will be used by cryptoKeys.encrypt.
	// with the resource name in the format: projects/*/locations/*/ekmConnections/*.
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/ekmConnections/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.ekmConnections/get
	if primary := cryptoKey.GetPrimary(); primary != nil {
		if protectionLevel := primary.GetProtectionLevel(); protectionLevel == kmspb.ProtectionLevel_EXTERNAL_VPC {
			if cryptoKeyBackend := cryptoKey.GetCryptoKeyBackend(); cryptoKeyBackend != "" {
				location := gcpshared.ExtractPathParam("locations", cryptoKeyBackend)
				ekmConnections := gcpshared.ExtractPathParam("ekmConnections", cryptoKeyBackend)
				if location != "" && ekmConnections != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   CloudKMSEKMConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(location, ekmConnections),
							Scope:  c.ProjectID(),
						},
						//Deleting the CryptoKeyBackend makes the CryptoKey non-functional
						//Deleting the CryptoKey doesn't affect the EKMConnection; EKM Connections are not owned by individual CryptoKeys.
						BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
					})
				}
			}
		}
	}

	// The IAM policy associated with this CryptoKey.
	// GET https://cloudkms.googleapis.com/v1/{resource=projects/*/locations/*/keyRings/*/cryptoKeys/*}:getIamPolicy
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys/getIamPolicy
	if name := cryptoKey.GetName(); name != "" {
		location := gcpshared.ExtractPathParam("locations", name)
		keyRings := gcpshared.ExtractPathParam("keyRings", name)
		cryptoKeys := gcpshared.ExtractPathParam("cryptoKeys", name)
		if location != "" && keyRings != "" && cryptoKeys != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   IAMPolicy.String(),
					Method: sdp.QueryMethod_GET,
					//TODO(Nauany): "":getIamPolicy" needs to be appended at the end of the URL, ensure team is aware
					Query: shared.CompositeLookupKey(location, keyRings, cryptoKeys),
					Scope: c.ProjectID(),
				},
				//Deleting the IAM Policy makes the CryptoKey non-functional
				//Deleting the CryptoKey deletes the IAM Policy
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	return sdpItem, nil
}
