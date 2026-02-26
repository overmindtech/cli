package dynamic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func Test_externalToSDP(t *testing.T) {
	type args struct {
		location       gcpshared.LocationInfo
		uniqueAttrKeys []string
		resp           map[string]any
		sdpAssetType   shared.ItemType
		nameSelector   string
	}
	testLocation := gcpshared.NewProjectLocation("test-project")
	tests := []struct {
		name    string
		args    args
		want    *sdp.Item
		wantErr bool
	}{
		{
			name: "ReturnsSDPItemWithCorrectAttributes",
			args: args{
				location:       testLocation,
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]any{
					"name":   "projects/test-project/locations/us-central1/instances/instance-1",
					"labels": map[string]any{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Scope:           testLocation.ToScope(),
				Tags:            map[string]string{"env": "prod"},
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("projects/test-project/locations/us-central1/instances/instance-1"),
							"foo":        structpb.NewStringValue("bar"),
							"uniqueAttr": structpb.NewStringValue("test-project|us-central1|instance-1"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsSDPItemWithCorrectAttributesWhenNameDoesNotHaveUniqueAttrKeys",
			args: args{
				location:       testLocation,
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]any{
					// There is name, but it does not include uniqueAttrKeys, expected to use the name as is.
					"name":   "instance-1",
					"labels": map[string]any{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Scope:           testLocation.ToScope(),
				Tags:            map[string]string{"env": "prod"},
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("instance-1"),
							"foo":        structpb.NewStringValue("bar"),
							"uniqueAttr": structpb.NewStringValue("instance-1"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsErrorWhenNameMissing",
			args: args{
				location:       testLocation,
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]any{
					"labels": map[string]any{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "UseCustomNameSelectorWhenProvided",
			args: args{
				location:       testLocation,
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]any{
					"instanceName": "instance-1",
					"labels":       map[string]any{"env": "prod"},
					"foo":          "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
				nameSelector: "instanceName", // This instructs to look for instanceName instead of name
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Scope:           testLocation.ToScope(),
				Tags:            map[string]string{"env": "prod"},
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"instanceName": structpb.NewStringValue("instance-1"),
							"foo":          structpb.NewStringValue("bar"),
							"uniqueAttr":   structpb.NewStringValue("instance-1"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsSDPItemWithEmptyLabels",
			args: args{
				location:       testLocation,
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]any{
					"name": "projects/test-project/locations/us-central1/instances/instance-2",
					"foo":  "baz",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("projects/test-project/locations/us-central1/instances/instance-2"),
							"foo":        structpb.NewStringValue("baz"),
							"uniqueAttr": structpb.NewStringValue("test-project|us-central1|instance-2"),
						},
					},
				},
				Scope: testLocation.ToScope(),
				Tags:  map[string]string{},
			},
			wantErr: false,
		},
	}
	linker := gcpshared.NewLinker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := externalToSDP(context.Background(), tt.args.location, tt.args.uniqueAttrKeys, tt.args.resp, tt.args.sdpAssetType, linker, tt.args.nameSelector)
			if (err != nil) != tt.wantErr {
				t.Errorf("externalToSDP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// got.Attributes = createAttr(t, tt.args.resp)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("externalToSDP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDescription_ReturnsSelectorWithNameWhenNoUniqueAttrKeys(t *testing.T) {
	got := getDescription(gcpshared.ComputeInstance, []string{})
	want := fmt.Sprintf("Get a %s by its \"name\"", gcpshared.ComputeInstance)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_getDescription_ReturnsSelectorWithUniqueAttrKeys(t *testing.T) {
	got := getDescription(gcpshared.BigQueryTable, []string{"datasets", "tables"})
	want := fmt.Sprintf("Get a %s by its \"datasets|tables\"", gcpshared.BigQueryTable)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_getDescription_ReturnsSelectorWithSingleUniqueAttrKey(t *testing.T) {
	got := getDescription(gcpshared.StorageBucket, []string{"buckets"})
	want := fmt.Sprintf("Get a %s by its \"name\"", gcpshared.StorageBucket)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_listDescription_ReturnsCorrectDescription(t *testing.T) {
	got := listDescription(gcpshared.ComputeInstance)
	want := "List all gcp-compute-instance"
	if got != want {
		t.Errorf("listDescription() got = %v, want %v", got, want)
	}
}

func Test_listDescription_HandlesEmptyScope(t *testing.T) {
	got := listDescription(gcpshared.BigQueryTable)
	want := "List all gcp-big-query-table"
	if got != want {
		t.Errorf("listDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_ReturnsSelectorWithMultipleKeys(t *testing.T) {
	got := searchDescription(gcpshared.ServiceDirectoryEndpoint, []string{"locations", "namespaces", "services", "endpoints"}, "")
	want := "Search for gcp-service-directory-endpoint by its \"locations|namespaces|services\""
	if got != want {
		t.Errorf("searchDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_ReturnsSelectorWithTwoKeys(t *testing.T) {
	got := searchDescription(gcpshared.BigQueryTable, []string{"datasets", "tables"}, "")
	want := "Search for gcp-big-query-table by its \"datasets\""
	if got != want {
		t.Errorf("searchDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_PanicsWithOneKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("searchDescription() did not panic with one unique attribute key; expected panic")
		}
	}()
	_ = searchDescription(gcpshared.StorageBucket, []string{"buckets"}, "")
}

func Test_searchDescription_WithCustomSearchDescription(t *testing.T) {
	customDesc := "Custom search description for gcp-service-directory-endpoint"
	got := searchDescription(gcpshared.ServiceDirectoryEndpoint, []string{"locations", "namespaces", "services", "endpoints"}, customDesc)
	if got != customDesc {
		t.Errorf("searchDescription() got = %v, want %v", got, customDesc)
	}
}

// TestStreamSDPItemsZeroItemsCachesNotFound verifies that when the API returns zero items,
// streamSDPItems caches NOTFOUND so a subsequent Lookup returns the cached error.
func TestStreamSDPItemsZeroItemsCachesNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"instances": []any{}})
	}))
	defer server.Close()

	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	location := gcpshared.NewProjectLocation("test-project")
	scope := location.ToScope()
	listMethod := sdp.QueryMethod_LIST

	a := Adapter{
		httpCli:              server.Client(),
		uniqueAttributeKeys:  []string{"instances"},
		sdpAssetType:         gcpshared.ComputeInstance,
		linker:               &gcpshared.Linker{},
		nameSelector:         "name",
		listResponseSelector: "",
	}
	stream := discovery.NewRecordingQueryResultStream()
	ck := sdpcache.CacheKeyFromParts(a.Name(), listMethod, scope, a.Type(), "")

	streamSDPItems(ctx, a, server.URL, location, stream, cache, ck)

	cacheHit, _, _, qErr, done := cache.Lookup(ctx, a.Name(), listMethod, scope, a.Type(), "", false)
	done()
	if !cacheHit {
		t.Fatal("expected cache hit after streamSDPItems with zero items")
	}
	if qErr == nil {
		t.Fatal("expected cached NOTFOUND error, got nil")
	}
	if qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("expected NOTFOUND, got %v", qErr.GetErrorType())
	}
}

// ListCachesNotFoundWithMemoryCache verifies that when List returns 0 items, NOTFOUND is cached
// and a second List returns 0 items from cache without calling the API again.
func TestListCachesNotFoundWithMemoryCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"instances": []any{}})
	}))
	defer server.Close()

	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	location := gcpshared.NewProjectLocation("test-project")
	scope := location.ToScope()

	listEndpointFunc := func(loc gcpshared.LocationInfo) (string, error) {
		return server.URL, nil
	}
	config := &AdapterConfig{
		Locations:            []gcpshared.LocationInfo{location},
		HTTPClient:           server.Client(),
		GetURLFunc:           func(string, gcpshared.LocationInfo) string { return "" },
		SDPAssetType:         gcpshared.ComputeInstance,
		SDPAdapterCategory:   sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Linker:               &gcpshared.Linker{},
		UniqueAttributeKeys:  []string{"instances"},
		NameSelector:         "name",
		ListResponseSelector: "",
	}
	adapter := NewListableAdapter(listEndpointFunc, config, cache)
	discAdapter := adapter.(discovery.Adapter)

	// Prove cache is empty before the first query
	cacheHit, _, _, _, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, scope, discAdapter.Type(), "", false)
	done()
	if cacheHit {
		t.Fatal("cache should be empty before first List")
	}

	items, err := adapter.List(ctx, scope, false)
	if err != nil {
		t.Fatalf("first List: unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("first List: expected 0 items, got %d", len(items))
	}

	// the not found error should be cached
	cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, scope, discAdapter.Type(), "", false)
	done()
	if !cacheHit {
		t.Fatal("expected cache hit for List after first call")
	}
	if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Fatalf("expected cached NOTFOUND for List, got %v", qErr)
	}

	items, err = adapter.List(ctx, scope, false)
	if err != nil {
		t.Fatalf("second List: unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("second List: expected 0 items, got %d", len(items))
	}
}

// SearchCachesNotFoundWithMemoryCache verifies that when Search returns 0 items, NOTFOUND is cached
// and a second Search returns 0 items from cache without calling the API again.
func TestSearchCachesNotFoundWithMemoryCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"instances": []any{}})
	}))
	defer server.Close()

	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	location := gcpshared.NewProjectLocation("test-project")
	scope := location.ToScope()
	query := "some-instance"

	searchEndpointFunc := func(q string, loc gcpshared.LocationInfo) string {
		return server.URL
	}
	config := &AdapterConfig{
		Locations:            []gcpshared.LocationInfo{location},
		HTTPClient:           server.Client(),
		GetURLFunc:           func(string, gcpshared.LocationInfo) string { return "" },
		SDPAssetType:         gcpshared.ComputeInstance,
		SDPAdapterCategory:   sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Linker:               &gcpshared.Linker{},
		UniqueAttributeKeys:  []string{"instances"},
		NameSelector:         "name",
		ListResponseSelector: "",
	}
	adapter := NewSearchableAdapter(searchEndpointFunc, config, "search by instances", cache)
	discAdapter := adapter.(discovery.Adapter)

	// Prove cache is empty before the first query
	cacheHit, _, _, _, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_SEARCH, scope, discAdapter.Type(), query, false)
	done()
	if cacheHit {
		t.Fatal("cache should be empty before first Search")
	}

	items, err := adapter.Search(ctx, scope, query, false)
	if err != nil {
		t.Fatalf("first Search: unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("first Search: expected 0 items, got %d", len(items))
	}

	// the not found error should be cached
	cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_SEARCH, scope, discAdapter.Type(), query, false)
	done()
	if !cacheHit {
		t.Fatal("expected cache hit for Search after first call")
	}
	if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Fatalf("expected cached NOTFOUND for Search, got %v", qErr)
	}

	items, err = adapter.Search(ctx, scope, query, false)
	if err != nil {
		t.Fatalf("second Search: unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("second Search: expected 0 items, got %d", len(items))
	}
}

// TestStreamSDPItemsExtractionErrorDoesNotCacheNotFound verifies that when the API returns
// items but extraction fails (e.g. missing required "name"), streamSDPItems does NOT cache NOTFOUND.
func TestStreamSDPItemsExtractionErrorDoesNotCacheNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Item without "name" causes externalToSDP to return error (ReturnsErrorWhenNameMissing).
		_ = json.NewEncoder(w).Encode(map[string]any{
			"instances": []any{
				map[string]any{"foo": "bar"},
			},
		})
	}))
	defer server.Close()

	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	location := gcpshared.NewProjectLocation("test-project")
	scope := location.ToScope()
	listMethod := sdp.QueryMethod_LIST

	a := Adapter{
		httpCli:              server.Client(),
		uniqueAttributeKeys:  []string{"instances"},
		sdpAssetType:         gcpshared.ComputeInstance,
		linker:               &gcpshared.Linker{},
		nameSelector:         "name",
		listResponseSelector: "",
	}
	stream := discovery.NewRecordingQueryResultStream()
	ck := sdpcache.CacheKeyFromParts(a.Name(), listMethod, scope, a.Type(), "")

	streamSDPItems(ctx, a, server.URL, location, stream, cache, ck)

	cacheHit, _, _, qErr, done := cache.Lookup(ctx, a.Name(), listMethod, scope, a.Type(), "", false)
	done()
	if cacheHit && qErr != nil && qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
		t.Error("extraction errors must not result in NOTFOUND being cached")
	}
}
