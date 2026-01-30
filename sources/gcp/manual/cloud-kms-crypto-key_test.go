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

func TestCloudKMSCryptoKey(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project-id"

	t.Run("Get_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with a CryptoKey item
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key",
			"uniqueAttr": "global|test-keyring|test-key",
		})
		_ = attrs.Set("uniqueAttr", "global|test-keyring|test-key")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKey.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
			Tags:            map[string]string{"env": "test"},
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKey.String(), "global|test-keyring|test-key")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "global|test-keyring|test-key", false)
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
		if uniqueAttr != "global|test-keyring|test-key" {
			t.Fatalf("Expected uniqueAttr 'global|test-keyring|test-key', got: %v", uniqueAttr)
		}

		// Verify tags
		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags())
		}
	})

	t.Run("Get_CacheMiss_NotFound", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with a NOTFOUND error to simulate item not existing
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "No resources found in Cloud Asset API",
		}
		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKey.String(), "global|test-keyring|nonexistent")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		// Get a non-existent item - should return NOTFOUND from cache
		_, err := adapter.Get(ctx, wrapper.Scopes()[0], "global|test-keyring|nonexistent", false)
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
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with CryptoKey items under SEARCH cache key (by keyRing)
		attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key-1",
			"uniqueAttr": "global|test-keyring|test-key-1",
		})
		_ = attrs1.Set("uniqueAttr", "global|test-keyring|test-key-1")

		attrs2, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key-2",
			"uniqueAttr": "global|test-keyring|test-key-2",
		})
		_ = attrs2.Set("uniqueAttr", "global|test-keyring|test-key-2")

		item1 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKey.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs1,
			Scope:           projectID,
		}
		item2 := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKey.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs2,
			Scope:           projectID,
		}

		// Search by location|keyRing
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKey.String(), "global|test-keyring")
		cache.StoreItem(ctx, item1, shared.DefaultCacheDuration, searchCacheKey)
		cache.StoreItem(ctx, item2, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "global|test-keyring", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("Search_CacheHit_Empty", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Store NOTFOUND error in cache to simulate empty result
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "No resources found in Cloud Asset API",
		}
		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSCryptoKey.String(), "global|empty-keyring")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "global|empty-keyring", false)
		if qErr != nil {
			t.Fatalf("Expected no error (empty search is valid), got: %v", qErr)
		}

		// Empty result is valid for SEARCH - should return empty slice, not error
		if len(items) != 0 {
			t.Fatalf("Expected 0 items (empty result), got: %d", len(items))
		}
	})

	t.Run("List_Unsupported", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
		}
	})

	t.Run("StaticTests", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with a CryptoKey item with linked queries
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key",
			"uniqueAttr": "global|test-keyring|test-key",
		})
		_ = attrs.Set("uniqueAttr", "global|test-keyring|test-key")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSCryptoKey.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   gcpshared.IAMPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  "global|test-keyring|test-key",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSKeyRing.String(),
						Method: sdp.QueryMethod_GET,
						Query:  "global|test-keyring",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  "global|test-keyring|test-key",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			},
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSCryptoKey.String(), "global|test-keyring|test-key")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSCryptoKey(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "global|test-keyring|test-key", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				ExpectedType:   gcpshared.IAMPolicy.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "global|test-keyring|test-key",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
			{
				ExpectedType:   gcpshared.CloudKMSKeyRing.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "global|test-keyring",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "global|test-keyring|test-key",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
		}

		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})
}
