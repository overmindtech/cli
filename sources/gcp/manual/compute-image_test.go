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

func TestComputeImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeImagesClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeImageWithLinks(projectID, "test-image", computepb.Image_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-image", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// sourceDisk link
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-source-disk",
					ExpectedScope:  fmt.Sprintf("%s.us-central1-a", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceSnapshot link
				{
					ExpectedType:   gcpshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-source-snapshot",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceImage link
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-source-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// licenses link (first license)
				{
					ExpectedType:   gcpshared.ComputeLicense.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-license-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// licenses link (second license)
				{
					ExpectedType:   gcpshared.ComputeLicense.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-license-2",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// rawDisk.source (GCS bucket) link
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("%s-raw-disk-bucket", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// imageEncryptionKey.kmsKeyName (CryptoKeyVersion) link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-image-key|test-version-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// imageEncryptionKey.kmsKeyServiceAccount link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-image-kms-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceImageEncryptionKey.kmsKeyName (CryptoKeyVersion) link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-source-image-key|test-version-source-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceImageEncryptionKey.kmsKeyServiceAccount link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-source-image-kms-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceSnapshotEncryptionKey.kmsKeyName (CryptoKeyVersion) link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-source-snapshot-key|test-version-source-snapshot",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// sourceSnapshotEncryptionKey.kmsKeyServiceAccount link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-source-snapshot-kms-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// deprecated.replacement link
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-replacement-image",
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

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Image_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Undefined",
				input:    computepb.Image_UNDEFINED_STATUS,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Deleting",
				input:    computepb.Image_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Failed",
				input:    computepb.Image_FAILED,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Pending",
				input:    computepb.Image_PENDING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Healthy",
				input:    computepb.Image_READY,
				expected: sdp.Health_HEALTH_OK,
			},
		}

		mockClient = mocks.NewMockComputeImagesClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeImage("test-instance", tc.input), nil)

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
		wrapper := manual.NewComputeImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeImageIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-1", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-2", computepb.Image_READY), nil)
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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeImageIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-1", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-2", computepb.Image_READY), nil)
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

func createComputeImage(imageName string, status computepb.Image_Status) *computepb.Image {
	return &computepb.Image{
		Name:   ptr.To(imageName),
		Labels: map[string]string{"env": "test"},
		Status: ptr.To(status.String()),
	}
}

func createComputeImageWithLinks(projectID, imageName string, status computepb.Image_Status) *computepb.Image {
	zone := "us-central1-a"
	sourceDiskURL := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/test-source-disk", projectID, zone)
	sourceSnapshotURL := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/snapshots/test-source-snapshot", projectID)
	sourceImageURL := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/test-source-image", projectID)
	replacementImageURL := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/test-replacement-image", projectID)

	return &computepb.Image{
		Name:           ptr.To(imageName),
		Labels:         map[string]string{"env": "test"},
		Status:         ptr.To(status.String()),
		SourceDisk:     &sourceDiskURL,
		SourceSnapshot: &sourceSnapshotURL,
		SourceImage:    &sourceImageURL,
		Licenses: []string{
			fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/licenses/test-license-1", projectID),
			fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/licenses/test-license-2", projectID),
		},
		RawDisk: &computepb.RawDisk{
			Source: ptr.To(fmt.Sprintf("gs://%s-raw-disk-bucket/raw-disk.tar.gz", projectID)),
		},
		ImageEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName:           ptr.To(fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-image-key/cryptoKeyVersions/test-version-image", projectID)),
			KmsKeyServiceAccount: ptr.To(fmt.Sprintf("projects/%s/serviceAccounts/test-image-kms-sa@%s.iam.gserviceaccount.com", projectID, projectID)),
		},
		SourceImageEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName:           ptr.To(fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-source-image-key/cryptoKeyVersions/test-version-source-image", projectID)),
			KmsKeyServiceAccount: ptr.To(fmt.Sprintf("projects/%s/serviceAccounts/test-source-image-kms-sa@%s.iam.gserviceaccount.com", projectID, projectID)),
		},
		SourceSnapshotEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName:           ptr.To(fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-source-snapshot-key/cryptoKeyVersions/test-version-source-snapshot", projectID)),
			KmsKeyServiceAccount: ptr.To(fmt.Sprintf("projects/%s/serviceAccounts/test-source-snapshot-kms-sa@%s.iam.gserviceaccount.com", projectID, projectID)),
		},
		Deprecated: &computepb.DeprecationStatus{
			Replacement: &replacementImageURL,
		},
	}
}
