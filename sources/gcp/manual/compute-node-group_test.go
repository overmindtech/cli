package manual_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeNodeGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeNodeGroupClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	testTemplateUrl := "https://www.googleapis.com/compute/v1/projects/test-project/regions/northamerica-northeast1/nodeTemplates/node-template-1"
	testTemplateUrl2 := "https://www.googleapis.com/compute/v1/projects/test-project/regions/northamerica-northeast1/nodeTemplates/node-template-2"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeNodeGroup("test-node-group", testTemplateUrl, computepb.NodeGroup_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-node-group", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNodeTemplate.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "node-template-1",
					ExpectedScope:  "test-project.northamerica-northeast1",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.NodeGroup_Status
			expected sdp.Health
		}

		testCases := []testCase{
			{
				name:     "Ready status",
				input:    computepb.NodeGroup_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Invalid status",
				input:    computepb.NodeGroup_INVALID,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Creating status",
				input:    computepb.NodeGroup_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting status",
				input:    computepb.NodeGroup_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeNodeGroup("test-ng", "test-temp", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-node-group", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Errorf("Expected health: %v, got: %v", tc.expected, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			query := item.GetLinkedItemQueries()[0].GetQuery().GetQuery()
			if !strings.Contains(query, "node-template") {
				t.Fatalf("Expected node-template in query, got: %s", query)
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		var errs []error
		mockItemHandler := func(item *sdp.Item) { items = append(items, item); wg.Done() }
		mockErrorHandler := func(err error) { errs = append(errs, err) }

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})

	t.Run("ListCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockComputeNodeGroupClient(ctrl)
		projectID := "cache-test-project"
		zone := "us-central1-a"
		scope := projectID + "." + zone

		mockAggIter := mocks.NewMockNodeGroupsScopedListPairIterator(ctrl)
		mockAggIter.EXPECT().Next().Return(compute.NodeGroupsScopedListPair{}, iterator.Done)
		mockListIter := mocks.NewMockComputeNodeGroupIterator(ctrl)
		mockListIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().AggregatedList(gomock.Any(), gomock.Any()).Return(mockAggIter).Times(1)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockListIter).Times(1)

		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		cache := sdpcache.NewMemoryCache()
		adapter := sources.WrapperToAdapter(wrapper, cache)
		discAdapter := adapter.(discovery.Adapter)
		listable := adapter.(discovery.ListableAdapter)

		// --- Scope "*" ---
		items, err := listable.List(ctx, "*", false)
		if err != nil {
			t.Fatalf("first List(*): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first List(*): expected 0 items, got %d", len(items))
		}
		cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, "*", discAdapter.Type(), "", false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for List(*)")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for List(*), got %v", qErr)
		}
		items, err = listable.List(ctx, "*", false)
		if err != nil {
			t.Fatalf("second List(*): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second List(*): expected 0 items, got %d", len(items))
		}

		// --- Specific scope ---
		items, err = listable.List(ctx, scope, false)
		if err != nil {
			t.Fatalf("first List(scope): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first List(scope): expected 0 items, got %d", len(items))
		}
		cacheHit, _, _, qErr, done = cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, scope, discAdapter.Type(), "", false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for List(scope)")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for List(scope), got %v", qErr)
		}
		items, err = listable.List(ctx, scope, false)
		if err != nil {
			t.Fatalf("second List(scope): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second List(scope): expected 0 items, got %d", len(items))
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		filterBy := testTemplateUrl

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, req *computepb.ListNodeGroupsRequest, opts ...any) *mocks.MockComputeNodeGroupIterator {
			fullList := []*computepb.NodeGroup{
				createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-3", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-4", testTemplateUrl, computepb.NodeGroup_READY),
			}

			expectedFilter := "nodeTemplate = " + filterBy
			if req.GetFilter() != expectedFilter {
				t.Fatalf("Expected filter to be %s, got: %s", expectedFilter, req.GetFilter())
			}

			mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)
			for _, nodeGroup := range fullList {
				if nodeGroup.GetNodeTemplate() == filterBy {
					mockComputeIterator.EXPECT().Next().Return(nodeGroup, nil)
				}
			}

			mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

			return mockComputeIterator
		})

		// [SPEC] Search filters by the node template URL. It will list and filter out
		// any node groups that are not using the given URL.

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], testTemplateUrl, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 1 of 4 are filtered out.
		if len(sdpItems) != 3 {
			t.Fatalf("Expected 3 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			attributes := item.GetAttributes()
			nodeTemplate, err := attributes.Get("node_template")
			if err != nil {
				t.Fatalf("Failed to get node_template attribute: %v", err)
			}

			if nodeTemplate != testTemplateUrl {
				t.Fatalf("Expected node_template to be %s, got: %s", testTemplateUrl, nodeTemplate)
			}
		}
	})

	t.Run("SearchCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockComputeNodeGroupClient(ctrl)
		projectID := "cache-test-project"
		zone := "us-central1-a"
		scope := projectID + "." + zone
		query := "https://www.googleapis.com/compute/v1/projects/cache-test-project/zones/us-central1-a/nodeTemplates/nonexistent-template"

		mockIter := mocks.NewMockComputeNodeGroupIterator(ctrl)
		mockIter.EXPECT().Next().Return(nil, iterator.Done)
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIter).Times(1)

		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		cache := sdpcache.NewMemoryCache()
		adapter := sources.WrapperToAdapter(wrapper, cache)
		discAdapter := adapter.(discovery.Adapter)
		searchable := adapter.(discovery.SearchableAdapter)

		items, err := searchable.Search(ctx, scope, query, false)
		if err != nil {
			t.Fatalf("first Search: unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first Search: expected 0 items, got %d", len(items))
		}

		cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_SEARCH, scope, discAdapter.Type(), query, false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for Search after first call")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for Search, got %v", qErr)
		}

		items, err = searchable.Search(ctx, scope, query, false)
		if err != nil {
			t.Fatalf("second Search: unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second Search: expected 0 items, got %d", len(items))
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		filterBy := testTemplateUrl

		mockClient.EXPECT().List(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, req *computepb.ListNodeGroupsRequest, opts ...any) *mocks.MockComputeNodeGroupIterator {
			fullList := []*computepb.NodeGroup{
				createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-3", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-4", testTemplateUrl, computepb.NodeGroup_READY),
			}

			expectedFilter := "nodeTemplate = " + filterBy
			if req.GetFilter() != expectedFilter {
				t.Fatalf("Expected filter to be %s, got: %s", expectedFilter, req.GetFilter())
			}

			mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)
			for _, nodeGroup := range fullList {
				if nodeGroup.GetNodeTemplate() == filterBy {
					mockComputeIterator.EXPECT().Next().Return(nodeGroup, nil)
				}
			}
			mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)
			return mockComputeIterator
		})

		var items []*sdp.Item
		var errs []error
		wg := &sync.WaitGroup{}
		wg.Add(3) // 3 items expected

		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		// Check if adapter supports search streaming
		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], testTemplateUrl, true, stream)
		wg.Wait()

		if len(errs) > 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 3 {
			t.Fatalf("Expected 3 items, got: %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
			attributes := item.GetAttributes()
			nodeTemplate, err := attributes.Get("node_template")
			if err != nil {
				t.Fatalf("Failed to get node_template attribute: %v", err)
			}
			if nodeTemplate != testTemplateUrl {
				t.Fatalf("Expected node_template to be %s, got: %s", testTemplateUrl, nodeTemplate)
			}
		}
	})
}

func createComputeNodeGroup(name, templateUrl string, status computepb.NodeGroup_Status) *computepb.NodeGroup {
	return &computepb.NodeGroup{
		Name:         new(name),
		NodeTemplate: new(templateUrl),
		Status:       new(status.String()),
	}
}
