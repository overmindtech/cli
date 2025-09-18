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
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeInstance(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeInstanceClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstance("test-instance", computepb.Instance_RUNNING), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.3",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-policy",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Instance_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Healthy",
				input:    computepb.Instance_RUNNING,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Terminated",
				input:    computepb.Instance_TERMINATED,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Stopped",
				input:    computepb.Instance_STOPPED,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Suspended",
				input:    computepb.Instance_SUSPENDED,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Provisioning",
				input:    computepb.Instance_PROVISIONING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Repairing",
				input:    computepb.Instance_REPAIRING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Staging",
				input:    computepb.Instance_STAGING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Stopping",
				input:    computepb.Instance_STOPPING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Suspending",
				input:    computepb.Instance_SUSPENDING,
				expected: sdp.Health_HEALTH_PENDING,
			},
		}

		mockClient = mocks.NewMockComputeInstanceClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeInstance(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstance("test-instance", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Fatalf("Expected health %s, got: %s", tc.expected, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeInstanceIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeInstance("test-instance-1", computepb.Instance_RUNNING), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeInstance("test-instance-2", computepb.Instance_RUNNING), nil)
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
		wrapper := manual.NewComputeInstance(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeInstanceIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeInstance("test-instance-1", computepb.Instance_RUNNING), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeInstance("test-instance-2", computepb.Instance_RUNNING), nil)
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
}

func createComputeInstance(instanceName string, status computepb.Instance_Status) *computepb.Instance {
	return &computepb.Instance{
		Name:   ptr.To(instanceName),
		Labels: map[string]string{"env": "test"},
		Disks: []*computepb.AttachedDisk{
			{
				DeviceName: ptr.To("test-disk"),
				Source:     ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/disks/test-instance"),
			},
		},
		NetworkInterfaces: []*computepb.NetworkInterface{
			{
				NetworkIP:   ptr.To("192.168.1.3"),
				Subnetwork:  ptr.To("projects/test-project-id/regions/us-central1/subnetworks/default"),
				Network:     ptr.To("global/networks/network"),
				Ipv6Address: ptr.To("2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
			},
		},
		Status: ptr.To(status.String()),
		ResourcePolicies: []string{
			"projects/test-project-id/regions/us-central1/resourcePolicies/test-policy",
		},
	}
}
