package manual_test

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sdp-go"
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

		adapter := sources.WrapperToAdapter(wrapper)

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
							ExpectedType:             manual.ComputeImage.String(),
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
							ExpectedType:             manual.ComputeSnapshot.String(),
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
							ExpectedType:             manual.ComputeInstantSnapshot.String(),
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
							ExpectedType:             manual.ComputeDisk.String(),
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
				ExpectedType:             manual.ComputeResourcePolicy.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-policy",
				ExpectedScope:            "test-project-id.us-central1",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}
			userTest := shared.QueryTest{
				ExpectedType:             manual.ComputeInstance.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-instance",
				ExpectedScope:            "test-project-id.us-central1-a",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
			}
			diskTypeTest := shared.QueryTest{
				ExpectedType:             manual.ComputeDiskType.String(),
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
				ExpectedType:             manual.ComputeResourcePolicy.String(),
				ExpectedMethod:           sdp.QueryMethod_GET,
				ExpectedQuery:            "test-consistency-group-policy",
				ExpectedScope:            "test-project-id.us-central1",
				ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					disk := createComputeDiskWithSource("test-disk", computepb.Disk_READY, tc.sourceType, tc.sourceValue)
					wrapper := manual.NewComputeDisk(mockClient, projectID, zone)
					adapter := sources.WrapperToAdapter(wrapper)

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
				adapter := sources.WrapperToAdapter(wrapper)

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
		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeDiskIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-1", computepb.Disk_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeDisk("test-disk-2", computepb.Disk_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		sdpItems, err := adapter.List(ctx, wrapper.Scopes()[0], true)
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
