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
	"k8s.io/utils/ptr"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeAddress(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeAddressClient(ctrl)
	projectID := "test-project-id"
	region := "us-central1"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeAddress("test-address"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-address", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  "test-project-id",
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  "test-project-id.us-central1",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.3",
					ExpectedScope:  "global",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeAddressIterator(ctrl)

		// Add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeAddress("test-address-1"), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeAddress("test-address-2"), nil)
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

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeAddressIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeAddress("test-address-1"), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeAddress("test-address-2"), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

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
		mockClient := mocks.NewMockComputeAddressClient(ctrl)
		projectID := "cache-test-project"
		region := "us-central1"
		scope := projectID + "." + region

		mockAggIter := mocks.NewMockAddressesScopedListPairIterator(ctrl)
		mockAggIter.EXPECT().Next().Return(compute.AddressesScopedListPair{}, iterator.Done)
		mockListIter := mocks.NewMockComputeAddressIterator(ctrl)
		mockListIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().AggregatedList(gomock.Any(), gomock.Any()).Return(mockAggIter).Times(1)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockListIter).Times(1)

		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
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

	t.Run("GetWithUsers", func(t *testing.T) {
		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		// Test with various user resource types
		users := []string{
			// Regional forwarding rule
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/forwardingRules/test-forwarding-rule", projectID, region),
			// Global forwarding rule
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules/test-global-forwarding-rule", projectID),
			// VM Instance
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/us-central1-a/instances/test-instance", projectID),
			// Target VPN Gateway
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetVpnGateways/test-vpn-gateway", projectID, region),
			// Router
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/test-router", projectID, region),
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeAddressWithUsers("test-address-with-users", users), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-address-with-users", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
				},
				// Subnetwork link
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				// IP address link
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.3",
					ExpectedScope:  "global",
				},
				// Regional forwarding rule link (from users)
				{
					ExpectedType:   gcpshared.ComputeForwardingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-forwarding-rule",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				// Global forwarding rule link (from users)
				{
					ExpectedType:   gcpshared.ComputeGlobalForwardingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-global-forwarding-rule",
					ExpectedScope:  projectID,
				},
				// Instance link (from users)
				{
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  fmt.Sprintf("%s.us-central1-a", projectID),
				},
				// Target VPN Gateway link (from users)
				{
					ExpectedType:   gcpshared.ComputeTargetVpnGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vpn-gateway",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				// Router link (from users)
				{
					ExpectedType:   gcpshared.ComputeRouter.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-router",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithIPCollection", func(t *testing.T) {
		wrapper := manual.NewComputeAddress(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		ipCollection := fmt.Sprintf("projects/%s/regions/%s/publicDelegatedPrefixes/test-prefix", projectID, region)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeAddressWithIPCollection("test-address-with-ip-collection", ipCollection), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-address-with-ip-collection", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
				},
				// Subnetwork link
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				// IP address link
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.3",
					ExpectedScope:  "global",
				},
				// Public Delegated Prefix link (from ipCollection)
				{
					ExpectedType:   gcpshared.ComputePublicDelegatedPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-prefix",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})
}

func createComputeAddress(addressName string) *computepb.Address {
	return &computepb.Address{
		Name:       ptr.To(addressName),
		Labels:     map[string]string{"env": "test"},
		Network:    ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/networks/network"),
		Subnetwork: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/subnetworks/default"),
		Address:    ptr.To("192.168.1.3"),
	}
}

func createComputeAddressWithUsers(addressName string, users []string) *computepb.Address {
	addr := createComputeAddress(addressName)
	addr.Users = users
	return addr
}

func createComputeAddressWithIPCollection(addressName string, ipCollection string) *computepb.Address {
	addr := createComputeAddress(addressName)
	addr.IpCollection = ptr.To(ipCollection)
	return addr
}
