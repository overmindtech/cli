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
	CloudKMSKeyRing = shared.NewItemType(gcpshared.GCP, gcpshared.CloudKMS, gcpshared.KeyRing)

	CloudKMSKeyRingLookupByName     = shared.NewItemTypeLookup("name", CloudKMSKeyRing)
	CloudKMSKeyRingLookupByLocation = shared.NewItemTypeLookup("location", CloudKMSKeyRing)
)

// cloudKMSKeyRingWrapper wraps the KMS KeyRing client for SDP adaptation.
type cloudKMSKeyRingWrapper struct {
	client gcpshared.CloudKMSKeyRingClient

	*gcpshared.ProjectBase
}

// NewCloudKMSKeyRing creates a new cloudKMSKeyRingWrapper.
func NewCloudKMSKeyRing(client gcpshared.CloudKMSKeyRingClient, projectID string) sources.SearchableWrapper {
	return &cloudKMSKeyRingWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			CloudKMSKeyRing,
		),
	}
}

// PotentialLinks returns the potential links for the kms key ring
func (c cloudKMSKeyRingWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		IAMPolicy,
	)
}

// TerraformMappings returns the Terraform mappings for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_kms_key_ring.name",
		},
	}
}

// GetLookups returns the lookups for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		CloudKMSKeyRingLookupByLocation,
		CloudKMSKeyRingLookupByName,
	}
}

// Get retrieves a KMS KeyRing by its name.
// The name must be in the format: projects/{PROJECT_ID}/locations/{LOCATION}/keyRings/{KEY_RING}
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings/get
func (c cloudKMSKeyRingWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location := queryParts[0]
	keyRingName := queryParts[1]

	name := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s",
		c.ProjectID(), location, keyRingName,
	)

	req := &kmspb.GetKeyRingRequest{
		Name: name,
	}

	keyRing, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpKeyRingToSDPItem(keyRing, location)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// SearchLookups returns the lookups for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSKeyRingLookupByLocation,
		},
	}
}

// Search searches KMS KeyRings and converts them to sdp.Items.
// Searchable adapter because location parameter needs to be passed as a queryPart.
// GET https://cloudkms.googleapis.com/v1/{parent=projects/*/locations/*}/keyRings
func (c cloudKMSKeyRingWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location := queryParts[0]
	parent := fmt.Sprintf("projects/%s/locations/%s", c.ProjectID(), location)

	it := c.client.Search(ctx, &kmspb.ListKeyRingsRequest{
		Parent: parent,
	})

	var items []*sdp.Item
	for {
		keyRing, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := c.gcpKeyRingToSDPItem(keyRing, queryParts[0])
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpKeyRingToSDPItem converts a GCP KeyRing to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings
func (c cloudKMSKeyRingWrapper) gcpKeyRingToSDPItem(keyRing *kmspb.KeyRing, location string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(keyRing)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            CloudKMSKeyRing.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
	}

	// CryptoKeys are associated with KeyRings, but no changes can be made to any of them that will affect the other.
	// Link will be skipped for now.

	// The IAM policy associated with this KeyRing.
	// GET https://cloudkms.googleapis.com/v1/{resource=projects/*/locations/*/keyRings/*}:getIamPolicy
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings/getIamPolicy
	if keyRingName := keyRing.GetName(); keyRingName != "" {
		keyRingID := gcpshared.ExtractPathParam("keyRings", keyRingName)
		if keyRingName := keyRing.GetName(); keyRingName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   IAMPolicy.String(),
					Method: sdp.QueryMethod_GET,
					//TODO(Nauany): "":getIamPolicy" needs to be appended at the end of the URL, ensure team is aware
					Query: shared.CompositeLookupKey(location, keyRingID),
					Scope: c.ProjectID(),
				},
				//Updating the IAM Policy makes the KeyRing non-functional
				//KeyRings cannot be deleted or updated
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true}})
		}

	}

	return sdpItem, nil
}
