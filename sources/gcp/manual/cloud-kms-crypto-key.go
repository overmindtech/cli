package manual

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var CloudKMSCryptoKeyLookupByName = shared.NewItemTypeLookup("name", gcpshared.CloudKMSCryptoKey)

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
			gcpshared.CloudKMSCryptoKey,
		),
	}
}

func (c cloudKMSCryptoKeyWrapper) IAMPermissions() []string {
	return []string{
		"cloudkms.cryptoKeys.get",
		"cloudkms.cryptoKeys.list",
	}
}

// PotentialLinks returns the potential links for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.CloudKMSEKMConnection,
		gcpshared.IAMPolicy,
		gcpshared.CloudKMSKeyRing,
	)
}

// TerraformMappings returns the Terraform mappings for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	// TODO: Revisit this when working on this ticket:
	// https://linear.app/overmind/issue/ENG-706/fix-terraform-mappings-for-crypto-key
	return nil
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

// SearchLookups returns the lookups for the CryptoKey wrapper.
func (c cloudKMSCryptoKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
			CloudKMSCryptoKeyRingLookupByName,
		},
	}
}

// Search searches KMS CryptoKeys and converts them to sdp.Items.
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

func (c cloudKMSCryptoKeyWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, queryParts ...string) {
	location := queryParts[0]
	keyRing := queryParts[1]

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s",
		c.ProjectID(), location, keyRing,
	)

	it := c.client.List(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: parent,
	})

	for {
		cryptoKey, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err))
			return
		}

		item, sdpErr := c.gcpCryptoKeyToSDPItem(cryptoKey)
		if sdpErr != nil {
			stream.SendError(sdpErr)
		}

		stream.SendItem(item)
	}
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

	// The unique attribute must be the same as the query parameter for the Get method.
	// Which is in the format: locations|keyRingName|cryptoKeyName
	// We will extract the path parameters from the CryptoKey name to create a unique lookup key.
	//
	// [CryptoKey][google.cloud.kms.v1.CryptoKey] in the format
	// `projects/*/locations/*/keyRings/*/cryptoKeys/*`.
	values := gcpshared.ExtractPathParams(cryptoKey.GetName(), "locations", "keyRings", "cryptoKeys")
	location := values[0]
	keyRing := values[1]
	cryptoKeyName := values[2]
	if len(values) != 3 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid CryptoKey name: %s", cryptoKey.GetName()),
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
		Type:            gcpshared.CloudKMSCryptoKey.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            cryptoKey.GetLabels(),
	}

	// The IAM policy associated with this CryptoKey.
	// GET https://cloudkms.googleapis.com/v1/{resource=projects/*/locations/*/keyRings/*/cryptoKeys/*}:getIamPolicy
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys/getIamPolicy
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.IAMPolicy.String(),
			Method: sdp.QueryMethod_GET,
			//TODO(Nauany): "":getIamPolicy" needs to be appended at the end of the URL, ensure team is aware
			Query: shared.CompositeLookupKey(location, keyRing, cryptoKeyName),
			Scope: c.ProjectID(),
		},
		//Deleting the IAM Policy makes the CryptoKey non-functional
		//Deleting the CryptoKey deletes the IAM Policy
		BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.CloudKMSKeyRing.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(location, keyRing),
			Scope:  c.ProjectID(),
		},
		//Deleting the KeyRing makes the CryptoKey non-functional
		//Deleting the CryptoKey does not affect the KeyRing; KeyRings are not owned by individual CryptoKeys.
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	// The resource name of the primary CryptoKeyVersion for this CryptoKey.
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions/get
	// Attribute link: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys#:~:text=keyRings/*/cryptoKeys/*.-,primary,-object%20(CryptoKeyVersion
	if primary := cryptoKey.GetPrimary(); primary != nil {
		if name := primary.GetName(); name != "" {
			keyVersionVals := gcpshared.ExtractPathParams(name, "locations", "keyRings", "cryptoKeys", "cryptoKeyVersions")
			if len(keyVersionVals) == 4 && keyVersionVals[0] != "" && keyVersionVals[1] != "" && keyVersionVals[2] != "" && keyVersionVals[3] != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(keyVersionVals...),
						Scope:  c.ProjectID(),
					},
					//If all versions of a crypto key are deleted it will become non-functional
					//CryptoKey can only be deleted if all versions are deleted
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				})
			}
		}

		// The EKM connection used by the key material, if applicable.
		// Only applicable if CryptoKeyVersions have a ProtectionLevel of EXTERNAL_VPC.
		// Primary is the CryptoKeyVersion that will be used by cryptoKeys.encrypt.
		// with the resource name in the format: projects/*/locations/*/ekmConnections/*.
		// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/ekmConnections/*}
		// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.ekmConnections/get
		if protectionLevel := primary.GetProtectionLevel(); protectionLevel == kmspb.ProtectionLevel_EXTERNAL_VPC {
			if cryptoKeyBackend := cryptoKey.GetCryptoKeyBackend(); cryptoKeyBackend != "" {
				backendVals := gcpshared.ExtractPathParams(cryptoKeyBackend, "locations", "ekmConnections")
				if len(backendVals) == 2 && backendVals[0] != "" && backendVals[1] != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.CloudKMSEKMConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(backendVals...),
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

	return sdpItem, nil
}
