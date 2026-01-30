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

func TestCloudKMSKeyRing(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project-id"

	t.Run("Get_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with a KeyRing item (simulating what the loader would do)
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring",
			"uniqueAttr": "us|test-keyring",
		})
		_ = attrs.Set("uniqueAttr", "us|test-keyring")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSKeyRing.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSKeyRing.String(), "us|test-keyring")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		// Create loader that won't need to make API calls since cache is populated
		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "us|test-keyring", false)
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
		if uniqueAttr != "us|test-keyring" {
			t.Fatalf("Expected uniqueAttr 'us|test-keyring', got: %v", uniqueAttr)
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
		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSKeyRing.String(), "us|nonexistent")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		// Get a non-existent item - should return NOTFOUND from cache
		_, err := adapter.Get(ctx, wrapper.Scopes()[0], "us|nonexistent", false)
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

	t.Run("List_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with KeyRing items under LIST cache key
		attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring-1",
			"uniqueAttr": "us|test-keyring-1",
		})
		_ = attrs1.Set("uniqueAttr", "us|test-keyring-1")

		attrs2, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring-2",
			"uniqueAttr": "us|test-keyring-2",
		})
		_ = attrs2.Set("uniqueAttr", "us|test-keyring-2")

		item1 := &sdp.Item{
			Type:            gcpshared.CloudKMSKeyRing.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs1,
			Scope:           projectID,
		}
		item2 := &sdp.Item{
			Type:            gcpshared.CloudKMSKeyRing.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs2,
			Scope:           projectID,
		}

		listCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_LIST, projectID, gcpshared.CloudKMSKeyRing.String(), "")
		cache.StoreItem(ctx, item1, shared.DefaultCacheDuration, listCacheKey)
		cache.StoreItem(ctx, item2, shared.DefaultCacheDuration, listCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		items, qErr := listable.List(ctx, wrapper.Scopes()[0], false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("List_CacheHit_Empty", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Store NOTFOUND error in cache to simulate empty result
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "No resources found in Cloud Asset API",
		}
		listCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_LIST, projectID, gcpshared.CloudKMSKeyRing.String(), "")
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, listCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		items, qErr := listable.List(ctx, wrapper.Scopes()[0], false)
		if qErr != nil {
			t.Fatalf("Expected no error (empty list is valid), got: %v", qErr)
		}

		// Empty result is valid for LIST - should return empty slice, not error
		if len(items) != 0 {
			t.Fatalf("Expected 0 items (empty result), got: %d", len(items))
		}
	})

	t.Run("Search_CacheHit", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with KeyRing items under SEARCH cache key (by location)
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring",
			"uniqueAttr": "us|test-keyring",
		})
		_ = attrs.Set("uniqueAttr", "us|test-keyring")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSKeyRing.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
		}

		searchCacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_SEARCH, projectID, gcpshared.CloudKMSKeyRing.String(), "us")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, searchCacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "us", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(items))
		}
	})

	t.Run("StaticTests", func(t *testing.T) {
		cache := sdpcache.NewCache(ctx)
		defer cache.Clear()

		// Pre-populate cache with a KeyRing item
		attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
			"name":       "projects/test-project-id/locations/us/keyRings/test-keyring",
			"uniqueAttr": "us|test-keyring",
		})
		_ = attrs.Set("uniqueAttr", "us|test-keyring")

		item := &sdp.Item{
			Type:            gcpshared.CloudKMSKeyRing.String(),
			UniqueAttribute: "uniqueAttr",
			Attributes:      attrs,
			Scope:           projectID,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   gcpshared.IAMPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  "us|test-keyring",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKey.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  "us|test-keyring",
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			},
		}

		cacheKey := sdpcache.CacheKeyFromParts("gcp-source", sdp.QueryMethod_GET, projectID, gcpshared.CloudKMSKeyRing.String(), "us|test-keyring")
		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

		loader := gcpshared.NewCloudKMSAssetLoader(nil, projectID, cache, "gcp-source", []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		wrapper := manual.NewCloudKMSKeyRing(loader, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, cache)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "us|test-keyring", false)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				ExpectedType:   gcpshared.IAMPolicy.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "us|test-keyring",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
			{
				ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "us|test-keyring",
				ExpectedScope:  "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  false,
					Out: true,
				},
			},
		}

		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})
}
