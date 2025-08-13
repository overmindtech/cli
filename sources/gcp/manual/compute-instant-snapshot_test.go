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
)

func TestComputeInstantSnapshot(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeInstantSnapshotsClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeInstantSnapshot(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstantSnapshot("test-snapshot", zone, computepb.InstantSnapshot_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-snapshot", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}
		// [SPEC] The default scope for disk is a combined zone and project id.
		if sdpItem.GetScope() != "test-project-id.us-central1-a" {
			t.Fatalf("Expected scope to be 'test-project-id.us-central1-a', got: %s", sdpItem.GetScope())
		}

		// [SPEC] Instant snapshots have one link: the source Disk.
		if len(sdpItem.GetLinkedItemQueries()) != 1 {
			t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
		}

		// [SPEC] Ensure Source Disk is linked
		linkedItem := sdpItem.GetLinkedItemQueries()[0]
		diskName := "test-disk"
		if linkedItem.GetQuery().GetType() != gcpshared.ComputeDisk.String() {
			t.Fatalf("Expected linked item type to be %s, got: %s", gcpshared.ComputeDisk, linkedItem.GetQuery().GetType())
		}

		if linkedItem.GetQuery().GetQuery() != diskName {
			t.Fatalf("Expected linked item query to be %s, got: %s", diskName, linkedItem.GetQuery().GetQuery())
		}

		if linkedItem.GetQuery().GetScope() != gcpshared.ZonalScope(projectID, zone) {
			t.Fatalf("Expected linked item scope to be %s, got: %s", gcpshared.ZonalScope(projectID, zone), linkedItem.GetQuery().GetScope())
		}
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.InstantSnapshot_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Undefined",
				input:    computepb.InstantSnapshot_UNDEFINED_STATUS,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Creating",
				input:    computepb.InstantSnapshot_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting",
				input:    computepb.InstantSnapshot_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Failed",
				input:    computepb.InstantSnapshot_FAILED,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Ready",
				input:    computepb.InstantSnapshot_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Unavailable",
				input:    computepb.InstantSnapshot_UNAVAILABLE,
				expected: sdp.Health_HEALTH_ERROR,
			},
		}

		mockClient = mocks.NewMockComputeInstantSnapshotsClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeInstantSnapshot(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstantSnapshot("test-snapshot", zone, tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-snapshot", true)
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
		wrapper := manual.NewComputeInstantSnapshot(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeInstantSnapshotIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeInstantSnapshot("test-snapshot-1", zone, computepb.InstantSnapshot_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeInstantSnapshot("test-snapshot-2", zone, computepb.InstantSnapshot_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		sdpItems, err := adapter.List(ctx, wrapper.Scopes()[0], true)
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
		wrapper := manual.NewComputeInstantSnapshot(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeInstantSnapshotIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeInstantSnapshot("test-snapshot-1", zone, computepb.InstantSnapshot_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeInstantSnapshot("test-snapshot-2", zone, computepb.InstantSnapshot_READY), nil)
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
		adapter.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})
}

func createComputeInstantSnapshot(snapshotName, zone string, status computepb.InstantSnapshot_Status) *computepb.InstantSnapshot {
	return &computepb.InstantSnapshot{
		Name:   ptr.To(snapshotName),
		Labels: map[string]string{"env": "test"},
		Status: ptr.To(status.String()),
		Zone:   ptr.To(zone),
		SourceDisk: ptr.To(
			"projects/test-project-id/zones/" + zone + "/disks/test-disk",
		),
		Architecture: ptr.To(computepb.InstantSnapshot_X86_64.String()),
	}
}
