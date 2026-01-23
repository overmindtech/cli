package manual_test

import (
	"context"
	"fmt"
	"strings"
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
)

func TestComputeBackendService(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGlobalClient := mocks.NewMockComputeBackendServiceClient(ctrl)
	mockRegionalClient := mocks.NewMockComputeRegionBackendServiceClient(ctrl)
	projectID := "test-project"

	t.Run("Get-Scope-Validation-Global", func(t *testing.T) {
		// Adapter configured for project-level only (global resources)
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Attempt to query a regional scope that wasn't configured
		unauthorizedScope := fmt.Sprintf("%s.us-central1", projectID)
		_, qErr := wrapper.Get(ctx, unauthorizedScope, "test-backend-service")

		// Should fail with NOSCOPE error since us-central1 wasn't configured
		if qErr == nil {
			t.Fatal("Expected error when querying unconfigured regional scope, got nil")
		}
		if qErr.GetErrorType() != sdp.QueryError_NOSCOPE {
			t.Errorf("Expected NOSCOPE error, got: %v (error: %s)", qErr.GetErrorType(), qErr.GetErrorString())
		}
	})

	t.Run("Get-Scope-Validation-Regional", func(t *testing.T) {
		// Adapter configured for us-west1 only
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, nil, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, "us-west1")})

		// Attempt to query us-central1 which wasn't configured
		unauthorizedScope := fmt.Sprintf("%s.us-central1", projectID)
		_, qErr := wrapper.Get(ctx, unauthorizedScope, "test-backend-service")

		// Should fail with NOSCOPE error since us-central1 wasn't configured
		if qErr == nil {
			t.Fatal("Expected error when querying unconfigured regional scope, got nil")
		}
		if qErr.GetErrorType() != sdp.QueryError_NOSCOPE {
			t.Errorf("Expected NOSCOPE error, got: %v (error: %s)", qErr.GetErrorType(), qErr.GetErrorString())
		}
	})

	t.Run("ListStream-Scope-Validation-Global", func(t *testing.T) {
		// Adapter configured for project-level only (global resources)
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Attempt to list from a regional scope that wasn't configured
		unauthorizedScope := fmt.Sprintf("%s.us-central1", projectID)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		cache := sdpcache.NewNoOpCache()

		wrapper.ListStream(ctx, stream, cache, sdpcache.CacheKey{}, unauthorizedScope)

		// Should fail with NOSCOPE error since us-central1 wasn't configured
		if len(errs) == 0 {
			t.Fatal("Expected error when listing from unconfigured regional scope, got none")
		}
		// The error should contain scope-related error message
		if len(errs) > 0 {
			// The first error should be a QueryError about scope
			expectedError := "scope"
			if err := errs[0]; err == nil || err.Error() == "" {
				t.Errorf("Expected error containing '%s', got nil or empty error", expectedError)
			} else if err := errs[0]; !strings.Contains(err.Error(), expectedError) {
				t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
			}
		}
	})

	t.Run("Get-Global", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeBackendService("test-backend-service"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List-Global", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockBackendServiceIterator := mocks.NewMockComputeBackendServiceIterator(ctrl)

		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockGlobalClient.EXPECT().List(ctx, gomock.Any()).Return(mockBackendServiceIterator)

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

	t.Run("ListStream-Global", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockBackendServiceIterator := mocks.NewMockComputeBackendServiceIterator(ctrl)

		// add mock implementation here
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service-1"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service-2"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockGlobalClient.EXPECT().List(ctx, gomock.Any()).Return(mockBackendServiceIterator)

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

	t.Run("GetWithHealthCheck", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Test with global health check
		healthCheckURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/healthChecks/test-health-check", projectID)
		backendService := createComputeBackendService("test-backend-service")
		backendService.HealthChecks = []string{healthCheckURL}

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(backendService, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeHealthCheck.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-health-check",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithRegionalHealthCheck", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Test with regional health check
		region := "us-central1"
		healthCheckURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/healthChecks/test-regional-health-check", projectID, region)
		backendService := createComputeBackendService("test-backend-service")
		backendService.HealthChecks = []string{healthCheckURL}

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(backendService, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeHealthCheck.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-regional-health-check",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInstanceGroup", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Test with unmanaged instance group
		zone := "us-central1-a"
		instanceGroupURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instanceGroups/test-instance-group", projectID, zone)
		backendService := createComputeBackendService("test-backend-service")
		backendService.Backends = []*computepb.Backend{
			{
				Group: ptr.To(instanceGroupURL),
			},
		}

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(backendService, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance-group",
					ExpectedScope:  zone,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithHAPolicy", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Test with HA Policy
		zone := "us-central1-a"
		backendGroupURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups/test-neg", projectID, zone)
		instanceName := "test-leader-instance"
		backendService := createComputeBackendService("test-backend-service")
		backendService.HaPolicy = &computepb.BackendServiceHAPolicy{
			Leader: &computepb.BackendServiceHAPolicyLeader{
				BackendGroup: ptr.To(backendGroupURL),
				NetworkEndpoint: &computepb.BackendServiceHAPolicyLeaderNetworkEndpoint{
					Instance: ptr.To(instanceName),
				},
			},
		}

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(backendService, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetworkEndpointGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-neg",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  instanceName,
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithRegion", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}, nil)

		// Test with region field (output-only, typically for regional backend services)
		region := "us-central1"
		regionURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s", projectID, region)
		backendService := createComputeBackendService("test-backend-service")
		backendService.Region = ptr.To(regionURL)

		mockGlobalClient.EXPECT().Get(ctx, gomock.Any()).Return(backendService, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeRegion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  region,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	// Regional backend service tests
	region := "us-central1"

	t.Run("Get-Regional", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, nil, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockRegionalClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeBackendService("test-regional-backend-service"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), "test-regional-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify the item has the correct type (should be ComputeBackendService, not ComputeRegionBackendService)
		if sdpItem.GetType() != gcpshared.ComputeBackendService.String() {
			t.Fatalf("Expected type to be '%s', got: %s", gcpshared.ComputeBackendService.String(), sdpItem.GetType())
		}

		// Verify the scope is regional
		if sdpItem.GetScope() != fmt.Sprintf("%s.%s", projectID, region) {
			t.Fatalf("Expected scope to be '%s.%s', got: %s", projectID, region, sdpItem.GetScope())
		}
	})

	t.Run("List-Regional", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockGlobalClient, mockRegionalClient, nil, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockBackendServiceIterator := mocks.NewMockComputeRegionBackendServiceIterator(ctrl)

		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-regional-backend-service-1"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-regional-backend-service-2"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockRegionalClient.EXPECT().List(ctx, gomock.Any()).Return(mockBackendServiceIterator)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, fmt.Sprintf("%s.%s", projectID, region), true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			// Verify each item has the correct type
			if item.GetType() != gcpshared.ComputeBackendService.String() {
				t.Fatalf("Expected type to be '%s', got: %s", gcpshared.ComputeBackendService.String(), item.GetType())
			}

			// Verify each item has the correct regional scope
			if item.GetScope() != fmt.Sprintf("%s.%s", projectID, region) {
				t.Fatalf("Expected scope to be '%s.%s', got: %s", projectID, region, item.GetScope())
			}

			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})
}

func createComputeBackendService(name string) *computepb.BackendService {
	return &computepb.BackendService{
		Name:               ptr.To(name),
		Network:            ptr.To("global/networks/network"),
		SecurityPolicy:     ptr.To("https://compute.googleapis.com/compute/v1/projects/test-project/global/securityPolicies/test-security-policy"),
		EdgeSecurityPolicy: ptr.To("https://compute.googleapis.com/compute/v1/projects/test-project/global/securityPolicies/test-edge-security-policy"),
		SecuritySettings: &computepb.SecuritySettings{
			ClientTlsPolicy: ptr.To("https://networksecurity.googleapis.com/v1/projects/test-project/locations/test-location/clientTlsPolicies/test-client-tls-policy"),
		},
		ServiceLbPolicy: ptr.To(" https://networkservices.googleapis.com/v1alpha1/name=projects/test-project/locations/test-location/serviceLbPolicies/test-service-lb-policy"),
		ServiceBindings: []string{
			"https://networkservices.googleapis.com/v1alpha1/projects/test-project/locations/test-location/serviceBindings/test-service-binding",
		},
	}
}
