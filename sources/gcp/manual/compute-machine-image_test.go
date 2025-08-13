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

func TestComputeMachineImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeMachineImageClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeMachineImage(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeMachineImage("test-machine-image", computepb.MachineImage_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-machine-image", true)
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
					ExpectedQuery:  "test-network",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
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
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
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
				}, {
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-machine-encryption-key",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  "test-project-id.us-central1-a",
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
			input    computepb.MachineImage_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Ready",
				input:    computepb.MachineImage_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Creating",
				input:    computepb.MachineImage_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting",
				input:    computepb.MachineImage_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Uploading",
				input:    computepb.MachineImage_UPLOADING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Invalid",
				input:    computepb.MachineImage_INVALID,
				expected: sdp.Health_HEALTH_ERROR,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeMachineImage(mockClient, projectID)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeMachineImage("test-machine-image", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-machine-image", true)
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
		wrapper := manual.NewComputeMachineImage(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeMachineImageIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-1", computepb.MachineImage_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-2", computepb.MachineImage_READY), nil)
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
		wrapper := manual.NewComputeMachineImage(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeMachineImageIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-1", computepb.MachineImage_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-2", computepb.MachineImage_READY), nil)
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

func createComputeMachineImage(imageName string, status computepb.MachineImage_Status) *computepb.MachineImage {
	return &computepb.MachineImage{
		Name:   ptr.To(imageName),
		Labels: map[string]string{"env": "test"},
		Status: ptr.To(status.String()),
		InstanceProperties: &computepb.InstanceProperties{
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Network:    ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/networks/test-network"),
					Subnetwork: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/subnetworks/test-subnetwork"),
				},
			},
			Disks: []*computepb.AttachedDisk{
				{
					Source: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/disks/test-disk"),
					DiskEncryptionKey: &computepb.CustomerEncryptionKey{
						KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-disk"),
					},
				},
			},
		},
		MachineImageEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-machine-encryption-key"),
		},
		SourceInstance: ptr.To("projects/test-project-id/zones/us-central1-a/instances/test-instance"),
	}
}
