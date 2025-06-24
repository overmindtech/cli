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
	CloudKMSCryptoKeyRingLookupByName     = shared.NewItemTypeLookup("name", gcpshared.CloudKMSKeyRing)
	CloudKMSCryptoKeyRingLookupByLocation = shared.NewItemTypeLookup("location", gcpshared.CloudKMSKeyRing)
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
			gcpshared.CloudKMSKeyRing,
		),
	}
}

// PotentialLinks returns the potential links for the kms key ring
func (c cloudKMSKeyRingWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.IAMPolicy,
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
		CloudKMSCryptoKeyRingLookupByLocation,
		CloudKMSCryptoKeyRingLookupByName,
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

	item, sdpErr := c.gcpKeyRingToSDPItem(keyRing)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// SearchLookups returns the lookups for the KeyRing wrapper.
func (c cloudKMSKeyRingWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CloudKMSCryptoKeyRingLookupByLocation,
		},
	}
}

// Search searches KMS KeyRings and converts them to sdp.Items.
// Searchable adapter because location parameter needs to be passed as a queryPart.
// GET https://cloudkms.googleapis.com/v1/{parent=projects/*/locations/*}/keyRings
func (c cloudKMSKeyRingWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	parent := fmt.Sprintf("projects/%s/locations/%s", c.ProjectID(), queryParts[0])

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

		item, sdpErr := c.gcpKeyRingToSDPItem(keyRing)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpKeyRingToSDPItem converts a GCP KeyRing to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings
func (c cloudKMSKeyRingWrapper) gcpKeyRingToSDPItem(keyRing *kmspb.KeyRing) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(keyRing)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	// The unique attribute must be the same as the query parameter for the Get method.
	// Which is in the format: locations|keyRingName
	// We will extract the path parameters from the KeyRing name to create a unique lookup key.
	//
	// Example KeyRing name: projects/{PROJECT_ID}/locations/{LOCATION}/keyRings/{KEY_RING}
	// Unique lookup key: locations|keyRingName
	// Extract the keyRingName from the KeyRing name.
	keyRingVals := gcpshared.ExtractPathParams(keyRing.GetName(), "locations", "keyRings")
	if len(keyRingVals) != 2 && keyRingVals[0] != "" && keyRingVals[1] != "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid KeyRing name: %s", keyRing.GetName()),
		}
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(keyRingVals...))
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to set unique attribute: %v", err),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.CloudKMSKeyRing.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
	}

	// The IAM policy associated with this KeyRing.
	// GET https://cloudkms.googleapis.com/v1/{resource=projects/*/locations/*/keyRings/*}:getIamPolicy
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings/getIamPolicy
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.IAMPolicy.String(),
			Method: sdp.QueryMethod_GET,
			//TODO(Nauany): "":getIamPolicy" needs to be appended at the end of the URL, ensure team is aware
			Query: shared.CompositeLookupKey(keyRingVals...),
			Scope: c.ProjectID(),
		},
		//Updating the IAM Policy makes the KeyRing non-functional
		//KeyRings cannot be deleted or updated
		BlastPropagation: &sdp.BlastPropagation{In: true, Out: true}})

	return sdpItem, nil
}
