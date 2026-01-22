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

func TestComputeMachineImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeMachineImageClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeMachineImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeMachineImage("test-machine-image", computepb.MachineImage_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-machine-image", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
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
				// Subnetwork link
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
				// Network Attachment link
				{
					ExpectedType:   gcpshared.ComputeNetworkAttachment.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network-attachment",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// IPv4 internal IP address
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// IPv6 internal address
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "2001:db8::1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// External IPv4 address (NAT IP)
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// External IPv6 address
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "2001:db8::2",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Disk source link
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
				// Disk encryption key
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
				// Source image link (SEARCH handles full URI)
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://www.googleapis.com/compute/v1/projects/test-project-id/global/images/test-source-image",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Source snapshot link
				{
					ExpectedType:   gcpshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-source-snapshot",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Source image encryption key
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-source-image",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Source snapshot encryption key
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
				// Service account link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sa@test-project-id.iam.gserviceaccount.com",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Accelerator type link
				{
					ExpectedType:   gcpshared.ComputeAcceleratorType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nvidia-tesla-k80",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Machine image encryption key
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-machine-encryption-key",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Source instance link
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
				// Saved disk link (from savedDisks)
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-saved-disk",
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
				wrapper := manual.NewComputeMachineImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		wrapper := manual.NewComputeMachineImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeMachineImageIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-1", computepb.MachineImage_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeMachineImage("test-machine-image-2", computepb.MachineImage_READY), nil)
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
		wrapper := manual.NewComputeMachineImage(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

func createComputeMachineImage(imageName string, status computepb.MachineImage_Status) *computepb.MachineImage {
	return &computepb.MachineImage{
		Name:   ptr.To(imageName),
		Labels: map[string]string{"env": "test"},
		Status: ptr.To(status.String()),
		InstanceProperties: &computepb.InstanceProperties{
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Network:           ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/networks/test-network"),
					Subnetwork:        ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/subnetworks/test-subnetwork"),
					NetworkAttachment: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/networkAttachments/test-network-attachment"),
					NetworkIP:         ptr.To("10.0.0.1"),
					Ipv6Address:       ptr.To("2001:db8::1"),
					AccessConfigs: []*computepb.AccessConfig{
						{
							NatIP: ptr.To("203.0.113.1"),
						},
					},
					Ipv6AccessConfigs: []*computepb.AccessConfig{
						{
							ExternalIpv6: ptr.To("2001:db8::2"),
						},
					},
				},
			},
			Disks: []*computepb.AttachedDisk{
				{
					Source: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/disks/test-disk"),
					DiskEncryptionKey: &computepb.CustomerEncryptionKey{
						KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-disk"),
					},
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						SourceImage:    ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/images/test-source-image"),
						SourceSnapshot: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/snapshots/test-source-snapshot"),
						SourceImageEncryptionKey: &computepb.CustomerEncryptionKey{
							KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-image"),
						},
						SourceSnapshotEncryptionKey: &computepb.CustomerEncryptionKey{
							KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-source-snapshot"),
						},
					},
				},
			},
			ServiceAccounts: []*computepb.ServiceAccount{
				{
					Email: ptr.To("test-sa@test-project-id.iam.gserviceaccount.com"),
				},
			},
			GuestAccelerators: []*computepb.AcceleratorConfig{
				{
					AcceleratorType:  ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/acceleratorTypes/nvidia-tesla-k80"),
					AcceleratorCount: ptr.To[int32](1),
				},
			},
		},
		MachineImageEncryptionKey: &computepb.CustomerEncryptionKey{
			KmsKeyName: ptr.To("projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-machine-encryption-key"),
		},
		SourceInstance: ptr.To("projects/test-project-id/zones/us-central1-a/instances/test-instance"),
		SavedDisks: []*computepb.SavedDisk{
			{
				SourceDisk: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/disks/test-saved-disk"),
			},
		},
	}
}
