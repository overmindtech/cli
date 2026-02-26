package manual_test

import (
	"context"
	"fmt"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeForwardingRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeForwardingRuleClient(ctrl)
	projectID := "test-project-id"
	region := "us-central1"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createForwardingRule("test-rule", projectID, region, "192.168.1.1"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  "test-project-id.us-central1",
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  "test-project-id",
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  "test-project-id.us-central1",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockForwardingRuleIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-1", projectID, region, "192.168.1.1"), nil)
		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-2", projectID, region, "192.168.1.2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

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
		}

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockForwardingRuleIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-1", projectID, region, "192.168.1.1"), nil)
		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-2", projectID, region, "192.168.1.2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockComputeForwardingRuleClient(ctrl)
		projectID := "cache-test-project"
		region := "us-central1"
		scope := projectID + "." + region

		mockAggIter := mocks.NewMockForwardingRulesScopedListPairIterator(ctrl)
		mockAggIter.EXPECT().Next().Return(compute.ForwardingRulesScopedListPair{}, iterator.Done)
		mockListIter := mocks.NewMockForwardingRuleIterator(ctrl)
		mockListIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().AggregatedList(gomock.Any(), gomock.Any()).Return(mockAggIter).Times(1)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockListIter).Times(1)

		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
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

	t.Run("GetWithTarget", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		// Test with TargetHttpProxy
		targetURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/test-target-proxy", projectID)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.Target = new(targetURL)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(forwardingRule, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputeTargetHttpProxy.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-target-proxy",
				ExpectedScope:  projectID,
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithBaseForwardingRule", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		baseForwardingRuleURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/forwardingRules/base-forwarding-rule", projectID, region)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.BaseForwardingRule = new(baseForwardingRuleURL)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(forwardingRule, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputeForwardingRule.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "base-forwarding-rule",
				ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithIPCollection", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		ipCollectionURL := fmt.Sprintf("projects/%s/regions/%s/publicDelegatedPrefixes/test-prefix", projectID, region)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.IpCollection = new(ipCollectionURL)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(forwardingRule, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputePublicDelegatedPrefix.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-prefix",
				ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithServiceDirectoryRegistrations", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		namespaceURL := fmt.Sprintf("projects/%s/locations/us-central1/namespaces/test-namespace", projectID)
		serviceName := "test-service"
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.ServiceDirectoryRegistrations = []*computepb.ForwardingRuleServiceDirectoryRegistration{
			{
				Namespace: new(namespaceURL),
				Service:   new(serviceName),
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(forwardingRule, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			// Add the new queries we're testing
			queryTests := append(baseQueries,
				shared.QueryTest{
					ExpectedType:   gcpshared.ServiceDirectoryNamespace.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1|test-namespace",
					ExpectedScope:  projectID,
				},
				shared.QueryTest{
					ExpectedType:   gcpshared.ServiceDirectoryService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1|test-namespace|test-service",
					ExpectedScope:  projectID,
				},
			)

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})
}

func createForwardingRule(name, projectID, region, ipAddress string) *computepb.ForwardingRule {
	return &computepb.ForwardingRule{
		Name:           new(name),
		IPAddress:      new(ipAddress),
		Labels:         map[string]string{"env": "test"},
		Network:        new(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/test-network", projectID)),
		Subnetwork:     new(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/test-subnetwork", projectID, region)),
		BackendService: new(fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/backendServices/backend-service", projectID, region)),
	}
}
