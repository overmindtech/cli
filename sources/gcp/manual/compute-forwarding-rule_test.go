package manual_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
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
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

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
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

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

	t.Run("GetWithTarget", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

		// Test with TargetHttpProxy
		targetURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/test-target-proxy", projectID)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.Target = ptr.To(targetURL)

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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputeTargetHttpProxy.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-target-proxy",
				ExpectedScope:  projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithBaseForwardingRule", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

		baseForwardingRuleURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/forwardingRules/base-forwarding-rule", projectID, region)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.BaseForwardingRule = ptr.To(baseForwardingRuleURL)

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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputeForwardingRule.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "base-forwarding-rule",
				ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithIPCollection", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

		ipCollectionURL := fmt.Sprintf("projects/%s/regions/%s/publicDelegatedPrefixes/test-prefix", projectID, region)
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.IpCollection = ptr.To(ipCollectionURL)

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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.ComputePublicDelegatedPrefix.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-prefix",
				ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithServiceDirectoryRegistrations", func(t *testing.T) {
		wrapper := manual.NewComputeForwardingRule(mockClient, projectID, region)

		namespaceURL := fmt.Sprintf("projects/%s/locations/us-central1/namespaces/test-namespace", projectID)
		serviceName := "test-service"
		forwardingRule := createForwardingRule("test-rule", projectID, region, "192.168.1.1")
		forwardingRule.ServiceDirectoryRegistrations = []*computepb.ForwardingRuleServiceDirectoryRegistration{
			{
				Namespace: ptr.To(namespaceURL),
				Service:   ptr.To(serviceName),
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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			// Add the new queries we're testing
			queryTests := append(baseQueries,
				shared.QueryTest{
					ExpectedType:   gcpshared.ServiceDirectoryNamespace.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1|test-namespace",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				shared.QueryTest{
					ExpectedType:   gcpshared.ServiceDirectoryService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1|test-namespace|test-service",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			)

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})
}

func createForwardingRule(name, projectID, region, ipAddress string) *computepb.ForwardingRule {
	return &computepb.ForwardingRule{
		Name:           ptr.To(name),
		IPAddress:      ptr.To(ipAddress),
		Labels:         map[string]string{"env": "test"},
		Network:        ptr.To(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/test-network", projectID)),
		Subnetwork:     ptr.To(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/test-subnetwork", projectID, region)),
		BackendService: ptr.To(fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/backendServices/backend-service", projectID, region)),
	}
}
