package manual_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
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

func TestComputeInstance(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeInstanceClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstance("test-instance", computepb.Instance_RUNNING), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
				wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

	t.Run("GetWithInitializeParams", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		// Test with sourceImage and sourceSnapshot in initializeParams
		sourceImageURL := fmt.Sprintf("projects/%s/global/images/test-image", projectID)
		sourceSnapshotURL := fmt.Sprintf("projects/%s/global/snapshots/test-snapshot", projectID)
		sourceImageKeyName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-image", projectID)
		sourceSnapshotKeyName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-snapshot", projectID)

		instance := createComputeInstance("test-instance", computepb.Instance_RUNNING)
		instance.Disks = []*computepb.AttachedDisk{
			{
				DeviceName: ptr.To("test-disk"),
				Source:     ptr.To(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/test-instance", projectID, zone)),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					SourceImage:    ptr.To(sourceImageURL),
					SourceSnapshot: ptr.To(sourceSnapshotURL),
					SourceImageEncryptionKey: &computepb.CustomerEncryptionKey{
						KmsKeyName: ptr.To(sourceImageKeyName),
					},
					SourceSnapshotEncryptionKey: &computepb.CustomerEncryptionKey{
						KmsKeyName: ptr.To(sourceSnapshotKeyName),
					},
				},
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(instance, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			// Add the new queries we're testing
			queryTests := append(baseQueries,
				shared.QueryTest{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				shared.QueryTest{
					ExpectedType:   gcpshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-snapshot",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				shared.QueryTest{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				shared.QueryTest{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|test-version-snapshot",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			)

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithDiskEncryptionKey", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		// Test with diskEncryptionKey (with version)
		diskKeyName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-key/cryptoKeyVersions/test-version-disk", projectID)

		instance := createComputeInstance("test-instance", computepb.Instance_RUNNING)
		instance.Disks = []*computepb.AttachedDisk{
			{
				DeviceName: ptr.To("test-disk"),
				Source:     ptr.To(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/test-instance", projectID, zone)),
				DiskEncryptionKey: &computepb.CustomerEncryptionKey{
					KmsKeyName: ptr.To(diskKeyName),
				},
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(instance, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "global|test-keyring|test-key|test-version-disk",
				ExpectedScope:  projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithDiskEncryptionKeyWithoutVersion", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		// Test with diskEncryptionKey (without version - should link to CryptoKey)
		diskKeyName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-keyring/cryptoKeys/test-key", projectID)

		instance := createComputeInstance("test-instance", computepb.Instance_RUNNING)
		instance.Disks = []*computepb.AttachedDisk{
			{
				DeviceName: ptr.To("test-disk"),
				Source:     ptr.To(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/test-instance", projectID, zone)),
				DiskEncryptionKey: &computepb.CustomerEncryptionKey{
					KmsKeyName: ptr.To(diskKeyName),
				},
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(instance, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "global|test-keyring|test-key",
				ExpectedScope:  projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithServiceAccount", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		// Test with service account email
		serviceAccountEmail := "test-service-account@test-project-id.iam.gserviceaccount.com"

		instance := createComputeInstance("test-instance", computepb.Instance_RUNNING)
		instance.ServiceAccounts = []*computepb.ServiceAccount{
			{
				Email: ptr.To(serviceAccountEmail),
			},
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(instance, nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Base queries that are always present
			baseQueries := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			// Add the new query we're testing
			queryTests := append(baseQueries, shared.QueryTest{
				ExpectedType:   gcpshared.IAMServiceAccount.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  serviceAccountEmail,
				ExpectedScope:  projectID,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("SupportsWildcardScope", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Check if adapter implements WildcardScopeAdapter
		if wildcardAdapter, ok := adapter.(discovery.WildcardScopeAdapter); ok {
			if !wildcardAdapter.SupportsWildcardScope() {
				t.Fatal("Expected SupportsWildcardScope to return true")
			}
		} else {
			t.Fatal("Expected adapter to implement WildcardScopeAdapter interface")
		}
	})

	t.Run("List with wildcard scope", func(t *testing.T) {
		zone1 := "us-central1-a"
		zone2 := "us-central1-b"
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{
			gcpshared.NewZonalLocation(projectID, zone1),
			gcpshared.NewZonalLocation(projectID, zone2),
		})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Create mock aggregated list iterator
		mockAggregatedIterator := mocks.NewMockInstancesScopedListPairIterator(ctrl)

		// Mock response for zone1
		mockAggregatedIterator.EXPECT().Next().Return(compute.InstancesScopedListPair{
			Key: "zones/us-central1-a",
			Value: &computepb.InstancesScopedList{
				Instances: []*computepb.Instance{
					createComputeInstance("instance-1-zone-a", computepb.Instance_RUNNING),
				},
			},
		}, nil)

		// Mock response for zone2
		mockAggregatedIterator.EXPECT().Next().Return(compute.InstancesScopedListPair{
			Key: "zones/us-central1-b",
			Value: &computepb.InstancesScopedList{
				Instances: []*computepb.Instance{
					createComputeInstance("instance-1-zone-b", computepb.Instance_RUNNING),
				},
			},
		}, nil)

		// Mock response for a zone not in our config (should be filtered)
		mockAggregatedIterator.EXPECT().Next().Return(compute.InstancesScopedListPair{
			Key: "zones/us-west1-a",
			Value: &computepb.InstancesScopedList{
				Instances: []*computepb.Instance{
					createComputeInstance("instance-west", computepb.Instance_RUNNING),
				},
			},
		}, nil)

		// End of iteration
		mockAggregatedIterator.EXPECT().Next().Return(compute.InstancesScopedListPair{}, iterator.Done)

		// Mock the AggregatedList method
		mockClient.EXPECT().AggregatedList(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...any) gcpshared.InstancesScopedListPairIterator {
				// Verify request parameters
				if req.GetProject() != projectID {
					t.Errorf("Expected project %s, got %s", projectID, req.GetProject())
				}
				if !req.GetReturnPartialSuccess() {
					t.Error("Expected ReturnPartialSuccess to be true")
				}
				return mockAggregatedIterator
			},
		)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		// Call List with wildcard scope
		sdpItems, err := listable.List(ctx, "*", true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should return only items from configured zones (zone-a and zone-b, not west1-a)
		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items (filtered), got: %d", len(sdpItems))
		}

		// Verify items have correct scopes
		scopesSeen := make(map[string]bool)
		for _, item := range sdpItems {
			scopesSeen[item.GetScope()] = true
		}

		expectedScopes := []string{
			fmt.Sprintf("%s.%s", projectID, zone1),
			fmt.Sprintf("%s.%s", projectID, zone2),
		}

		for _, expectedScope := range expectedScopes {
			if !scopesSeen[expectedScope] {
				t.Errorf("Expected to see scope %s in results", expectedScope)
			}
		}
	})

	t.Run("List with specific scope still works", func(t *testing.T) {
		wrapper := manual.NewComputeInstance(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockComputeIterator := mocks.NewMockComputeInstanceIterator(ctrl)

		// Mock normal per-zone List behavior
		mockComputeIterator.EXPECT().Next().Return(createComputeInstance("test-instance", computepb.Instance_RUNNING), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		// Call List with specific scope (not wildcard)
		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
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
				Network:     ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/global/networks/network"),
				Ipv6Address: ptr.To("2001:0db8:85a3:0000:0000:8a2e:0370:7334"),
			},
		},
		Status: ptr.To(status.String()),
		ResourcePolicies: []string{
			"projects/test-project-id/regions/us-central1/resourcePolicies/test-policy",
		},
	}
}
