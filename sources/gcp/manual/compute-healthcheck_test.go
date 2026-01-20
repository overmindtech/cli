package manual_test

import (
	"context"
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

func TestComputeHealthCheck(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeHealthCheckClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createHealthCheck("test-healthcheck"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-healthcheck", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// [SPEC] The default scope is the project ID.
		if sdpItem.GetScope() != "test-project-id" {
			t.Fatalf("Expected scope to be 'test-project-id', got: %s", sdpItem.GetScope())
		}

		// [SPEC] TCP HealthChecks have no linked items (no host field).
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Fatalf("Expected 0 linked item queries for TCP health check, got: %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("GetWithHTTPHealthCheck", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		httpHealthCheck := createHTTPHealthCheck("test-http-healthcheck", "example.com")
		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(httpHealthCheck, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-http-healthcheck", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// DNS name link from HTTP health check host field
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "example.com",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithHTTPSHealthCheckWithIP", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		httpsHealthCheck := createHTTPSHealthCheck("test-https-healthcheck", "192.168.1.100")
		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(httpsHealthCheck, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-https-healthcheck", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// IP address link from HTTPS health check host field
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.100",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithSourceRegions", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		healthCheckWithRegions := createHealthCheckWithSourceRegions("test-healthcheck-regions", []string{"us-central1", "us-east1", "europe-west1"})
		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(healthCheckWithRegions, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-healthcheck-regions", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Region links from sourceRegions array
				{
					ExpectedType:   gcpshared.ComputeRegion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeRegion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-east1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeRegion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "europe-west1",
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

	t.Run("GetWithRegion", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		regionalHealthCheck := createRegionalHealthCheck("test-regional-healthcheck", "us-central1")
		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(regionalHealthCheck, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-regional-healthcheck", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Region link from region field (output only, for regional health checks)
				{
					ExpectedType:   gcpshared.ComputeRegion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us-central1",
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

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeHealthCheckIter := mocks.NewMockComputeHealthCheckIterator(ctrl)

		// Mock out items listed from the API.
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-1"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-2"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeHealthCheckIter)

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
		wrapper := manual.NewComputeHealthCheck(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeHealthCheckIter := mocks.NewMockComputeHealthCheckIterator(ctrl)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-1"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-2"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeHealthCheckIter)

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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})
}

func createHealthCheck(healthCheckName string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("TCP"),
		TcpHealthCheck: &computepb.TCPHealthCheck{
			Port: ptr.To(int32(80)),
		},
	}
}

func createHTTPHealthCheck(healthCheckName, host string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("HTTP"),
		HttpHealthCheck: &computepb.HTTPHealthCheck{
			Port:        ptr.To(int32(80)),
			Host:        ptr.To(host),
			RequestPath: ptr.To("/"),
		},
	}
}

func createHTTPSHealthCheck(healthCheckName, host string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("HTTPS"),
		HttpsHealthCheck: &computepb.HTTPSHealthCheck{
			Port:        ptr.To(int32(443)),
			Host:        ptr.To(host),
			RequestPath: ptr.To("/"),
		},
	}
}

func createHealthCheckWithSourceRegions(healthCheckName string, regions []string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(30)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("TCP"),
		TcpHealthCheck: &computepb.TCPHealthCheck{
			Port: ptr.To(int32(80)),
		},
		SourceRegions: regions,
	}
}

func createRegionalHealthCheck(healthCheckName, region string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("TCP"),
		TcpHealthCheck: &computepb.TCPHealthCheck{
			Port: ptr.To(int32(80)),
		},
		Region: ptr.To(region),
	}
}
