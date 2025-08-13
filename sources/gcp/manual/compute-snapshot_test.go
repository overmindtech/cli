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

func TestComputeSnapshot(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeSnapshotsClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeSnapshot(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeSnapshot("test-snapshot", computepb.Snapshot_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-snapshot", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeLicense.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-license",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeInstantSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instant-snapshot",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-source-snapshot",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-source-disk",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-source-snapshot-schedule-policy",
					ExpectedScope:            "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Snapshot_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Undefined",
				input:    computepb.Snapshot_UNDEFINED_STATUS,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Creating",
				input:    computepb.Snapshot_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting",
				input:    computepb.Snapshot_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Failed",
				input:    computepb.Snapshot_FAILED,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Ready",
				input:    computepb.Snapshot_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Uploading",
				input:    computepb.Snapshot_UPLOADING,
				expected: sdp.Health_HEALTH_PENDING,
			},
		}

		mockClient = mocks.NewMockComputeSnapshotsClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeSnapshot(mockClient, projectID)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeSnapshot("test-snapshot", tc.input), nil)

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
		wrapper := manual.NewComputeSnapshot(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeSnapshotIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeSnapshot("test-snapshot-1", computepb.Snapshot_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeSnapshot("test-snapshot-2", computepb.Snapshot_READY), nil)
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
		wrapper := manual.NewComputeSnapshot(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeSnapshotIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeSnapshot("test-snapshot-1", computepb.Snapshot_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeSnapshot("test-snapshot-2", computepb.Snapshot_READY), nil)
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

func createComputeSnapshot(snapshotName string, status computepb.Snapshot_Status) *computepb.Snapshot {
	return &computepb.Snapshot{
		Name:                  ptr.To(snapshotName),
		Labels:                map[string]string{"env": "test"},
		Status:                ptr.To(status.String()),
		SourceInstantSnapshot: ptr.To("projects/test-project-id/zones/us-central1-a/instantSnapshots/test-instant-snapshot"),
		StorageLocations:      []string{"us-central1"},
		Licenses:              []string{"projects/test-project-id/global/licenses/test-license"},
		SourceDiskEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-disk"),
		},
		SourceDisk: ptr.To("projects/test-project-id/zones/us-central1-a/disks/test-disk"),
		SourceInstantSnapshotEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-snapshot"),
			RawKey:     ptr.To("test-key"),
		},
		SourceSnapshotSchedulePolicy: ptr.To("projects/test-project-id/regions/us-central1/resourcePolicies/test-source-snapshot-schedule-policy"),
	}
}
