package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudKMSCryptoKeyVersion(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project-id"

	t.Run("Get_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		// Pre-populate cache with a CryptoKeyVersion item
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/1",
			"uniqueAttr": "us|test-keyring|test-key|1",
			"state":      "ENABLED",
		})
		_ = attrs.Set("uniqueAttr", "us|test-keyring|test-key|1")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us|test-keyring|test-key|1")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "us|test-keyring|test-key|1", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem == nil {
			t.Fatalf("Expected item, got nil")
		}

		uniqueAttr, err := sdpItem.GetAttributes().Get("uniqueAttr")
		if err != nil {
			t.Fatalf("Failed to get uniqueAttr: %v", err)
		}
		if uniqueAttr != "us|test-keyring|test-key|1" {
			t.Fatalf("Expected uniqueAttr 'us|test-keyring|test-key|1', got: %v", uniqueAttr)
		}

		// Verify health
		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Fatalf("Expected health HEALTH_OK, got: %v", sdpItem.GetHealth())
		}
	})

	t.Run("Get_CacheMiss_NotFound", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		// Pre-populate cache with a NOTFOUND error to simulate item not existing
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "No resources found in Cloud Asset API",
		}
		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us|test-keyring|test-key|99")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		// Get a non-existent item - should return NOTFOUND from cache
		_, err := adapter.Get(ctx, wrapper.Scopes()[0], "us|test-keyring|test-key|99", false)
		if err == nil {
			t.Fatalf("Expected NOTFOUND error, got nil")
		}
		var qErr *sdp.QueryError
		if !errors.As(err, &qErr) {
			t.Fatalf("Expected QueryError, got: %T - %v", err, err)
		}
		if qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("Expected NOTFOUND error type, got: %v", qErr.GetErrorType())
		}
	})

	t.Run("Search_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		// Pre-populate cache with CryptoKeyVersion items under SEARCH cache key (by cryptoKey)
		attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/1",
			"uniqueAttr": "us|test-keyring|test-key|1",
		})
		_ = attrs1.Set("uniqueAttr", "us|test-keyring|test-key|1")

		attrs2, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/2",
			"uniqueAttr": "us|test-keyring|test-key|2",
		})
		_ = attrs2.Set("uniqueAttr", "us|test-keyring|test-key|2")

		item1 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs1,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
		}
		item2 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs2,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_WARNING.Enum(),
		}

		// Search by location|keyRing|cryptoKey
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us|test-keyring|test-key")
		cache.StoreItem(ctx, item1, shared.DefaultCacheDuration, searchCacheKey)
		cache.StoreItem(ctx, item2, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "us|test-keyring|test-key", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("Search_CacheHit_Empty", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		// Store NOTFOUND error in cache to simulate empty result
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "No resources found in Cloud Asset API",
		}
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us|test-keyring|empty-key")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "us|test-keyring|empty-key", false)
		if qErr != nil {
			t.Fatalf("Expected no error (empty search is valid), got: %v", qErr)
		}

		// Empty result is valid for SEARCH - should return empty slice, not error
		if len(items) != 0 {
			t.Fatalf("Expected 0 items (empty result), got: %d", len(items))
		}
	})

	t.Run("List_Unsupported", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
		}
	})

	t.Run("Search_TerraformFormat", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with CryptoKeyVersion items under SEARCH cache key (by cryptoKey)
		attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us-central1/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/1",
			"uniqueAttr": "us-central1|my-keyring|my-key|1",
		})
		_ = attrs1.Set("uniqueAttr", "us-central1|my-keyring|my-key|1")

		attrs2, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us-central1/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/2",
			"uniqueAttr": "us-central1|my-keyring|my-key|2",
		})
		_ = attrs2.Set("uniqueAttr", "us-central1|my-keyring|my-key|2")

		item1 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs1,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
		}
		item2 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs2,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
		}

		// Search by location|keyRing|cryptoKey (what the terraform format will be converted to)
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us-central1|my-keyring|my-key")
		cache.StoreItem(ctx, item1, shared.DefaultCacheDuration, searchCacheKey)
		cache.StoreItem(ctx, item2, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		// Use terraform-style path format
		terraformStyleQuery := "projects/test-project-id/locations/us-central1/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/1"

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], terraformStyleQuery, false)
		if qErr != nil {
			t.Fatalf("Expected no error with terraform format, got: %v", qErr)
		}

		// Verify we got at least one item back
		if len(items) == 0 {
			t.Fatalf("Expected at least 1 item with terraform format, got: %d", len(items))
		}

		// Verify the items have the expected unique attributes
		foundVersion1 := false
		for _, item := range items {
			uniqueAttr, err := item.GetAttributes().Get("uniqueAttr")
			if err == nil && (uniqueAttr == "us-central1|my-keyring|my-key|1" || uniqueAttr == "us-central1|my-keyring|my-key|2") {
				if uniqueAttr == "us-central1|my-keyring|my-key|1" {
					foundVersion1 = true
				}
			}
		}

		if !foundVersion1 {
			t.Fatalf("Expected to find version 1 in results")
		}
	})

	t.Run("Search_LegacyPipeFormat", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with CryptoKeyVersion items
		attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/europe-west1/keyRings/prod-keyring/cryptoKeys/prod-key/cryptoKeyVersions/1",
			"uniqueAttr": "europe-west1|prod-keyring|prod-key|1",
		})
		_ = attrs1.Set("uniqueAttr", "europe-west1|prod-keyring|prod-key|1")

		item1 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs1,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
		}

		// Search by location|keyRing|cryptoKey (legacy format)
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "europe-west1|prod-keyring|prod-key")
		cache.StoreItem(ctx, item1, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		// Use legacy pipe-separated format with multiple query parts
		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "europe-west1|prod-keyring|prod-key", false)
		if qErr != nil {
			t.Fatalf("Expected no error with legacy format, got: %v", qErr)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item with legacy format, got: %d", len(items))
		}
	})

	t.Run("StaticTests", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		defer cache.Clear()

		// Pre-populate cache with a CryptoKeyVersion item with linked queries
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/1",
			"uniqueAttr": "us|test-keyring|test-key|1",
		})
		_ = attrs.Set("uniqueAttr", "us|test-keyring|test-key|1")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKeyVersion.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
			Health:          sdp.Health_HEALTH_OK.Enum(),
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  "us|test-keyring|test-key",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKeyVersion.String(), "us|test-keyring|test-key|1")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKeyVersion(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "us|test-keyring|test-key|1", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "us|test-keyring|test-key",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		}

		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})
}
