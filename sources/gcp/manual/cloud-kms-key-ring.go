package manual

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/sourcegraph/conc/iter"
	"google.golang.org/api/iterator"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"

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

// cloudKMSKeyRingWrapper wraps the KMS KeyRing client for SDP adaptation.
type cloudKMSKeyRingWrapper struct {
	client gcpshared.CloudKMSKeyRingClient

	*gcpshared.ProjectBase
}

// NewCloudKMSKeyRing creates a new cloudKMSKeyRingWrapper.
func NewCloudKMSKeyRing(client gcpshared.CloudKMSKeyRingClient, locations []gcpshared.LocationInfo) sources.SearchableListableWrapper {
	return &cloudKMSKeyRingWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CloudKMSKeyRing,
		),
	}
}

func (c cloudKMSKeyRingWrapper) IAMPermissions() []string {
	return []string{
		"cloudkms.keyRings.get",
		"cloudkms.keyRings.list",
		"cloudkms.locations.list",
	}
}

func (c cloudKMSKeyRingWrapper) PredefinedRole() string {
	return "roles/cloudkms.viewer"
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
func (c cloudKMSKeyRingWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	location := queryParts[0]
	keyRingName := queryParts[1]

	name := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s",
		loc.ProjectID, location, keyRingName,
	)

	req := &kmspb.GetKeyRingRequest{
		Name: name,
	}

	keyRing, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpKeyRingToSDPItem(keyRing, loc)
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
func (c cloudKMSKeyRingWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", loc.ProjectID, queryParts[0])

	it := c.client.Search(ctx, &kmspb.ListKeyRingsRequest{
		Parent: parent,
	})

	var items []*sdp.Item
	for {
		keyRing, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpKeyRingToSDPItem(keyRing, loc)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// SearchStream streams the search results for KMS KeyRings.
func (c cloudKMSKeyRingWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", loc.ProjectID, queryParts[0])

	it := c.client.Search(ctx, &kmspb.ListKeyRingsRequest{
		Parent: parent,
	})

	for {
		keyRing, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpKeyRingToSDPItem(keyRing, loc)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// List lists all KMS KeyRings across all locations in the project.
// It first lists all available KMS locations, then lists key rings from each location in parallel.
func (c cloudKMSKeyRingWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	// List all available KMS locations
	parent := fmt.Sprintf("projects/%s", loc.ProjectID)
	locationIt := c.client.ListLocations(ctx, &locationpb.ListLocationsRequest{
		Name: parent,
	})

	var locationIDs []string
	for {
		location, iterErr := locationIt.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		// Extract location ID from the full location name
		// Format: projects/{PROJECT_ID}/locations/{LOCATION_ID}
		locationName := location.GetName()
		parts := strings.Split(locationName, "/")
		if len(parts) >= 4 && parts[len(parts)-2] == "locations" {
			locationIDs = append(locationIDs, parts[len(parts)-1])
		}
	}

	if len(locationIDs) == 0 {
		return []*sdp.Item{}, nil
	}

	// Use conc/iter to parallelize key ring listing across locations (10x concurrency)
	type result struct {
		items []*sdp.Item
		err   *sdp.QueryError
	}

	mapper := iter.Mapper[string, result]{
		MaxGoroutines: 10,
	}

	results, mapErr := mapper.MapErr(locationIDs, func(locationIDPtr *string) (result, error) {
		locationID := *locationIDPtr
		var locationItems []*sdp.Item

		// Use Search to list key rings in this location
		searchItems, searchErr := c.Search(ctx, scope, locationID)
		if searchErr != nil {
			return result{err: searchErr}, errors.New(searchErr.GetErrorString())
		}

		locationItems = append(locationItems, searchItems...)
		return result{items: locationItems}, nil
	})

	// Check for mapping errors
	if mapErr != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: mapErr.Error(),
		}
	}

	// Aggregate all results
	var allItems []*sdp.Item
	for _, res := range results {
		if res.err != nil {
			return nil, res.err
		}
		allItems = append(allItems, res.items...)
	}

	return allItems, nil
}

// ListStream streams all KMS KeyRings across all locations in the project.
// It first lists all available KMS locations, then streams key rings from each location in parallel.
func (c cloudKMSKeyRingWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	loc, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	// List all available KMS locations
	parent := fmt.Sprintf("projects/%s", loc.ProjectID)
	locationIt := c.client.ListLocations(ctx, &locationpb.ListLocationsRequest{
		Name: parent,
	})

	var locationIDs []string
	for {
		location, iterErr := locationIt.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		// Extract location ID from the full location name
		// Format: projects/{PROJECT_ID}/locations/{LOCATION_ID}
		locationName := location.GetName()
		parts := strings.Split(locationName, "/")
		if len(parts) >= 4 && parts[len(parts)-2] == "locations" {
			locationIDs = append(locationIDs, parts[len(parts)-1])
		}
	}

	if len(locationIDs) == 0 {
		return
	}

	// Use SearchStream for each location in parallel
	// We'll use a wait group to coordinate streaming from multiple locations
	var wg sync.WaitGroup
	for _, locationID := range locationIDs {
		wg.Add(1)
		go func(locID string) {
			defer wg.Done()
			c.SearchStream(ctx, stream, cache, cacheKey, scope, locID)
		}(locationID)
	}
	wg.Wait()
}

// gcpKeyRingToSDPItem converts a GCP KeyRing to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings
func (c cloudKMSKeyRingWrapper) gcpKeyRingToSDPItem(keyRing *kmspb.KeyRing, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
	if len(keyRingVals) != 2 || keyRingVals[0] == "" || keyRingVals[1] == "" {
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
		Scope:           location.ToScope(),
	}

	// The IAM policy associated with this KeyRing.
	// GET https://cloudkms.googleapis.com/v1/{resource=projects/*/locations/*/keyRings/*}:getIamPolicy
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings/getIamPolicy
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.IAMPolicy.String(),
			Method: sdp.QueryMethod_GET,
			// TODO(Nauany): "":getIamPolicy" needs to be appended at the end of the URL, ensure team is aware
			Query: shared.CompositeLookupKey(keyRingVals...),
			Scope: location.ProjectID,
		},
		// Updating the IAM Policy makes the KeyRing non-functional
		// KeyRings cannot be deleted or updated
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	// The KMS CryptoKeys associated with this KeyRing.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.CloudKMSCryptoKey.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(keyRingVals[0], keyRingVals[1]), // location|keyRingName
			Scope:  location.ProjectID,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false,
			Out: true,
		},
	})

	return sdpItem, nil
}
