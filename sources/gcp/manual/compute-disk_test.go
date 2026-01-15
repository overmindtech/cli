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
)

func TestComputeDisk(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeDiskClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeDisk("test-disk", computepb.Disk_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		expectedTag := "test"
		actualTag := sdpItem.GetTags()["env"]
		if actualTag != expectedTag {
			t.Fatalf("Expected tag 'env=%s', got: %v", expectedTag, actualTag)
		}

		t.Run("StaticTests", func(t *testing.T) {
			type staticTestCase struct {
				name           string
				sourceType     string
				sourceValue    string
				expectedLinked shared.QueryTests
			}
			cases := []staticTestCase{
				{
					name:        "SourceImage",
					sourceType:  "image",
					sourceValue: "projects/test-project-id/global/images/test-image",
					expectedLinked: shared.QueryTests{
						{
							ExpectedType:             gcpshared.ComputeImage.String(),
							ExpectedMethod:           sdp.QueryMethod_GET,
							ExpectedQuery:            "test-image",
							ExpectedScope:            "test-project-id",
							ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
						},
					},
				},
				{
					name:        "SourceSnapshot",
					sourceType:  "snapshot",
					sourceValue: "projects/test-project-id/global/snapshots/test-snapshot",
					expectedLinked: shared.QueryTests{
						{
							ExpectedType:             gcpshared.ComputeSnapshot.String(),
							ExpectedMethod:           sdp.QueryMethod_GET,
							ExpectedQuery:            "test-snapshot",
							ExpectedScope:            "test-project-id",
							ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
						},
					},
				},
				{
					name:        "SourceInstantSnapshot",
					sourceType:  "instantSnapshot",
					sourceValue: "projects/test-project-id/zones/us-central1-a/instantSnapshots/test-instant-snapshot",
					expectedLinked: shared.QueryTests{
						{
							ExpectedType:             gcpshared.ComputeInstantSnapshot.String(),
							ExpectedMethod:           sdp.QueryMethod_GET,
							ExpectedQuery:            "test-instant-snapshot",
							ExpectedScope:            "test-project-id.us-central1-a",
							ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
						},
					},
				},
				{
					name:        "SourceDisk",
					sourceType:  "disk",
					sourceValue: "projects/test-project-id/zones/us-central1-a/disks/source-disk",
					expectedLinked: shared.QueryTests{
						{
							ExpectedType:             gcpshared.ComputeDisk.String(),
							ExpectedMethod:           sdp.QueryMethod_GET,
							ExpectedQuery:            "source-disk",
							ExpectedScope:            "test-project-id.us-central1-a",
							ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
						},
					},
				},
			}

			// These are always present
			resourcePolicyTest := shared.QueryTest{
				ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-policy",
				ExpectedScope:            "test-project-id.us-central1",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			userTest := shared.QueryTest{
				ExpectedType:             gcpshared.ComputeInstance.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-instance",
				ExpectedScope:            "test-project-id.us-central1-a",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
			}
			diskTypeTest := shared.QueryTest{
				ExpectedType:             gcpshared.ComputeDiskType.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "pd-standard",
				ExpectedScope:            "test-project-id.us-central1-a",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			diskEncryptionKeyTest := shared.QueryTest{
				ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "global|test-keyring|test-key|test-version-source-disk",
				ExpectedScope:            "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			sourceImageEncryptionKeyTest := shared.QueryTest{
				ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "global|test-keyring|test-key|test-version-source-image",
				ExpectedScope:            "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			sourceSnapshotEncryptionKeyTest := shared.QueryTest{
				ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "global|test-keyring|test-key|test-version-source-snapshot",
				ExpectedScope:            "test-project-id",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			sourceConsistencyGroupPolicy := shared.QueryTest{
				ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-consistency-group-policy",
				ExpectedScope:            "test-project-id.us-central1",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					disk := createComputeDiskWithSource("test-disk", computepb.Disk_READY, tc.sourceType, tc.sourceValue)
					wrapper := manual.NewComputeDisk(mockClient, projectID, zone)
					adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

					// Mock the Get call to return our disk
					mockClient.EXPECT().Get(ctx, gomock.Any()).Return(disk, nil)

					sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
					if qErr != nil {
						t.Fatalf("Expected no error, got: %v", qErr)
					}

					// Compose expected queries for this source type
					expectedQueries := append(tc.expectedLinked, resourcePolicyTest, userTest, diskTypeTest, diskEncryptionKeyTest, sourceImageEncryptionKeyTest, sourceSnapshotEncryptionKeyTest, sourceConsistencyGroupPolicy)
					shared.RunStaticTests(t, adapter, sdpItem, expectedQueries)
				})
			}
		})

	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Disk_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Ready",
				input:    computepb.Disk_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Creating",
				input:    computepb.Disk_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Restoring",
				input:    computepb.Disk_RESTORING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting",
				input:    computepb.Disk_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Failed",
				input:    computepb.Disk_FAILED,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Unavailable",
				input:    computepb.Disk_UNAVAILABLE,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Unknown",
				input:    computepb.Disk_UNDEFINED_STATUS,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
		}

		mockClient = mocks.NewMockComputeDiskClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeDisk(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeDisk("test-disk", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Fatalf("Expected health %s, got: %s (input: %s)", tc.expected, sdpItem.GetHealth(), tc.input)
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeDiskIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-1", computepb.Disk_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-2", computepb.Disk_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

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

		expectedCount := 2
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
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
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeDiskIterator(ctrl)
		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-1", computepb.Disk_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-2", computepb.Disk_READY), nil)
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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("GetWithSourceStorageObject", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)

		// Test with gs:// URI format
		sourceStorageObject := "gs://test-bucket/path/to/image.tar.gz"
		disk := createComputeDisk("test-disk", computepb.Disk_READY)
		disk.SourceStorageObject = ptr.To(sourceStorageObject)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(disk, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeInstance.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-instance",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
				{
					ExpectedType:             gcpshared.ComputeDiskType.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "pd-standard",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-disk",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-snapshot",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-group-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeImage.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:             gcpshared.StorageBucket.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-bucket",
				ExpectedScope:            projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithStoragePool", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)

		storagePoolURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools/test-storage-pool", projectID, zone)
		disk := createComputeDisk("test-disk", computepb.Disk_READY)
		disk.StoragePool = ptr.To(storagePoolURL)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(disk, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present (same as above)
			baseQueries := shared.QueryTests{
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeInstance.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-instance",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
				{
					ExpectedType:             gcpshared.ComputeDiskType.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "pd-standard",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-disk",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-snapshot",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-group-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeImage.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:             gcpshared.ComputeStoragePool.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-storage-pool",
				ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithAsyncPrimaryDisk", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)

		primaryDiskURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/disks/primary-disk", projectID, zone)
		consistencyGroupPolicyURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/us-central1/resourcePolicies/test-consistency-policy", projectID)
		disk := createComputeDisk("test-disk", computepb.Disk_READY)
		disk.AsyncPrimaryDisk = &computepb.DiskAsyncReplication{
			Disk:                      ptr.To(primaryDiskURL),
			ConsistencyGroupPolicy:    ptr.To(consistencyGroupPolicyURL),
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(disk, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeInstance.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-instance",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
				{
					ExpectedType:             gcpshared.ComputeDiskType.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "pd-standard",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-disk",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-snapshot",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-group-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeImage.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			}

			// Add the new queries we're testing
			queryTests := append(baseQueries,
				shared.QueryTest{
					ExpectedType:             gcpshared.ComputeDisk.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "primary-disk",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				shared.QueryTest{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			)

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithAsyncSecondaryDisks", func(t *testing.T) {
		wrapper := manual.NewComputeDisk(mockClient, projectID, zone)

		secondaryDisk1URL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/disks/secondary-disk-1", projectID, zone)
		secondaryDisk2URL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/disks/secondary-disk-2", projectID, zone)
		consistencyGroupPolicyURL := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/us-central1/resourcePolicies/test-consistency-policy", projectID)
		disk := createComputeDisk("test-disk", computepb.Disk_READY)
		disk.AsyncSecondaryDisks = map[string]*computepb.DiskAsyncReplicationList{
			"secondary-disk-1": {
				AsyncReplicationDisk: &computepb.DiskAsyncReplication{
					Disk:                      ptr.To(secondaryDisk1URL),
					ConsistencyGroupPolicy:    ptr.To(consistencyGroupPolicyURL),
				},
			},
			"secondary-disk-2": {
				AsyncReplicationDisk: &computepb.DiskAsyncReplication{
					Disk: ptr.To(secondaryDisk2URL),
				},
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(disk, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-disk", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeInstance.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-instance",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
				{
					ExpectedType:             gcpshared.ComputeDiskType.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "pd-standard",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-disk",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "global|test-keyring|test-key|test-version-source-snapshot",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-group-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				{
					ExpectedType:             gcpshared.ComputeImage.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-image",
					ExpectedScope:            projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			}

			// Add the new queries we're testing
			queryTests := append(baseQueries,
				shared.QueryTest{
					ExpectedType:             gcpshared.ComputeDisk.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "secondary-disk-1",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				shared.QueryTest{
					ExpectedType:             gcpshared.ComputeResourcePolicy.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-consistency-policy",
					ExpectedScope:            fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
				shared.QueryTest{
					ExpectedType:             gcpshared.ComputeDisk.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "secondary-disk-2",
					ExpectedScope:            fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				},
			)

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

}

func createComputeDisk(diskName string, status computepb.Disk_Status) *computepb.Disk {
	return createComputeDiskWithSource(diskName, status, "image", "projects/test-project-id/global/images/test-image")
}

// createComputeDiskWithSource creates a Disk with only the specified source field set.
// sourceType can be "image", "snapshot", "instantSnapshot", or "disk".
// sourceValue is the value to set for the source field.
func createComputeDiskWithSource(diskName string, status computepb.Disk_Status, sourceType, sourceValue string) *computepb.Disk {
	disk := &computepb.Disk{
		Name:             ptr.To(diskName),
		Labels:           map[string]string{"env": "test"},
		Type:             ptr.To("projects/test-project-id/zones/us-central1-a/diskTypes/pd-standard"),
		Status:           ptr.To(status.String()),
		ResourcePolicies: []string{"projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"},
		Users:            []string{"projects/test-project-id/zones/us-central1-a/instances/test-instance"},
		DiskEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-disk"),
			RawKey:     ptr.To("test-key"),
		},
		SourceImageEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-image"),
			RawKey:     ptr.To("test-key"),
		},
		SourceSnapshotEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-snapshot"),
			RawKey:     ptr.To("test-key"),
		},
		SourceConsistencyGroupPolicy: ptr.To("projects/test-project-id/regions/us-central1/resourcePolicies/test-consistency-group-policy"),
	}

	switch sourceType {
	case "image":
		disk.SourceImage = ptr.To(sourceValue)
	case "snapshot":
		disk.SourceSnapshot = ptr.To(sourceValue)
	case "instantSnapshot":
		disk.SourceInstantSnapshot = ptr.To(sourceValue)
	case "disk":
		disk.SourceDisk = ptr.To(sourceValue)
	default:
		// Default to image if unknown type
		disk.SourceImage = ptr.To("projects/test-project-id/global/images/test-image")
	}

	return disk
}
