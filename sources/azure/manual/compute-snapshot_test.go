package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeSnapshot(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		snapshotName := "test-snapshot"
		snapshot := createAzureSnapshot(snapshotName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeSnapshot.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeSnapshot, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != snapshotName {
			t.Errorf("Expected unique attribute value %s, got %s", snapshotName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got %s", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.DiskAccessID
					ExpectedType:   azureshared.ComputeDiskAccess.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk-access",
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
				{
					// Properties.Encryption.DiskEncryptionSetID
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-des",
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
				{
					// Properties.CreationData.SourceResourceID (disk)
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "source-disk",
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithSnapshotSource", func(t *testing.T) {
		snapshotName := "test-snapshot-from-snapshot"
		snapshot := createAzureSnapshotFromSnapshot(snapshotName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.CreationData.SourceResourceID (snapshot)
					ExpectedType:   azureshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "source-snapshot",
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithSourceURI", func(t *testing.T) {
		snapshotName := "test-snapshot-from-blob"
		snapshot := createAzureSnapshotFromBlobURI(snapshotName)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.CreationData.SourceURI → Storage Account
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "teststorageaccount",
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
				{
					// Properties.CreationData.SourceURI → Blob Container
					ExpectedType:   azureshared.StorageBlobContainer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("teststorageaccount", "vhds"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,				},
				{
					// Properties.CreationData.SourceURI → HTTP
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://teststorageaccount.blob.core.windows.net/vhds/my-disk.vhd",
					ExpectedScope:  "global",				},
				{
					// Properties.CreationData.SourceURI → DNS
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "teststorageaccount.blob.core.windows.net",
					ExpectedScope:  "global",				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithSourceURIUsingIPHost", func(t *testing.T) {
		snapshotName := "test-snapshot-from-ip-blob"
		snapshot := createAzureSnapshotFromIPBlobURI(snapshotName)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.CreationData.SourceURI → HTTP
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://10.0.0.1/vhds/my-disk.vhd",
					ExpectedScope:  "global",				},
				{
					// Properties.CreationData.SourceURI → IP (host is IP address)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

		// Verify no DNS link was emitted for the IP host
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				t.Error("Expected no DNS link when SourceURI host is an IP address")
			}
		}
	})

	t.Run("GetWithEncryptionIPHosts", func(t *testing.T) {
		snapshotName := "test-snapshot-encryption-ip"
		snapshot := createAzureSnapshotWithEncryptionIPHosts(snapshotName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		foundSecretIPLink := false
		foundKeyIPLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == stdlib.NetworkIP.String() {
				if link.GetQuery().GetQuery() == "10.0.0.2" {
					foundSecretIPLink = true
				}
				if link.GetQuery().GetQuery() == "10.0.0.3" {
					foundKeyIPLink = true
				}
				if link.GetQuery().GetScope() != "global" {
					t.Errorf("Expected IP scope 'global', got %s", link.GetQuery().GetScope())
				}
				if link.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected IP method GET, got %s", link.GetQuery().GetMethod())
				}
			}
			if link.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				t.Error("Expected no DNS link when SecretURL/KeyURL hosts are IP addresses")
			}
		}

		if !foundSecretIPLink {
			t.Error("Expected to find IP link for SecretURL host 10.0.0.2")
		}
		if !foundKeyIPLink {
			t.Error("Expected to find IP link for KeyURL host 10.0.0.3")
		}
	})

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		snapshotName := "test-snapshot-cross-rg"
		snapshot := createAzureSnapshotWithCrossResourceGroupLinks(snapshotName, subscriptionID)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		foundDiskAccessLink := false
		foundDiskLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.ComputeDiskAccess.String() {
				foundDiskAccessLink = true
				expectedScope := subscriptionID + ".other-rg"
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected DiskAccess scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
			if link.GetQuery().GetType() == azureshared.ComputeDisk.String() {
				foundDiskLink = true
				expectedScope := subscriptionID + ".disk-rg"
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected Disk scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
		}

		if !foundDiskAccessLink {
			t.Error("Expected to find Disk Access link")
		}
		if !foundDiskLink {
			t.Error("Expected to find Disk link")
		}
	})

	t.Run("GetWithoutLinks", func(t *testing.T) {
		snapshotName := "test-snapshot-no-links"
		snapshot := createAzureSnapshotWithoutLinks(snapshotName)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, snapshotName, nil).Return(
			armcompute.SnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], snapshotName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		snapshot1 := createAzureSnapshot("test-snapshot-1", subscriptionID, resourceGroup)
		snapshot2 := createAzureSnapshot("test-snapshot-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockPager := newMockSnapshotsPager(ctrl, []*armcompute.Snapshot{snapshot1, snapshot2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		snapshot1 := createAzureSnapshot("test-snapshot-1", subscriptionID, resourceGroup)
		snapshot2 := createAzureSnapshot("test-snapshot-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockPager := newMockSnapshotsPager(ctrl, []*armcompute.Snapshot{snapshot1, snapshot2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

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

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		snapshot1 := createAzureSnapshot("test-snapshot-1", subscriptionID, resourceGroup)
		snapshotNilName := &armcompute.Snapshot{
			Name:     nil,
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockPager := newMockSnapshotsPager(ctrl, []*armcompute.Snapshot{snapshot1, snapshotNilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("snapshot not found")

		mockClient := mocks.NewMockSnapshotsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-snapshot", nil).Return(
			armcompute.SnapshotsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-snapshot", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent snapshot, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockSnapshotsClient(ctrl)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting snapshot with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSnapshotsClient(ctrl)

		wrapper := manual.NewComputeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when getting snapshot with insufficient query parts, but got nil")
		}
	})
}

// createAzureSnapshot creates a mock Azure Snapshot with linked resources for testing
func createAzureSnapshot(name, subscriptionID, resourceGroup string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			DiskAccessID:      to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskAccesses/test-disk-access"),
			Encryption: &armcompute.Encryption{
				DiskEncryptionSetID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-des"),
			},
			CreationData: &armcompute.CreationData{
				CreateOption:    to.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/source-disk"),
			},
		},
	}
}

// createAzureSnapshotFromSnapshot creates a mock Snapshot that was copied from another snapshot
func createAzureSnapshotFromSnapshot(name, subscriptionID, resourceGroup string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			CreationData: &armcompute.CreationData{
				CreateOption:    to.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/snapshots/source-snapshot"),
			},
		},
	}
}

// createAzureSnapshotFromBlobURI creates a mock Snapshot imported from a blob URI
func createAzureSnapshotFromBlobURI(name string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionImport),
				SourceURI:    to.Ptr("https://teststorageaccount.blob.core.windows.net/vhds/my-disk.vhd"),
			},
		},
	}
}

// createAzureSnapshotFromIPBlobURI creates a mock Snapshot imported from a blob URI with an IP address host
func createAzureSnapshotFromIPBlobURI(name string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionImport),
				SourceURI:    to.Ptr("https://10.0.0.1/vhds/my-disk.vhd"),
			},
		},
	}
}

// createAzureSnapshotWithEncryptionIPHosts creates a mock Snapshot with encryption settings using IP-based SecretURL and KeyURL
func createAzureSnapshotWithEncryptionIPHosts(name, subscriptionID, resourceGroup string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionEmpty),
			},
			EncryptionSettingsCollection: &armcompute.EncryptionSettingsCollection{
				Enabled: to.Ptr(true),
				EncryptionSettings: []*armcompute.EncryptionSettingsElement{
					{
						DiskEncryptionKey: &armcompute.KeyVaultAndSecretReference{
							SourceVault: &armcompute.SourceVault{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-vault"),
							},
							SecretURL: to.Ptr("https://10.0.0.2/secrets/my-secret/version1"),
						},
						KeyEncryptionKey: &armcompute.KeyVaultAndKeyReference{
							SourceVault: &armcompute.SourceVault{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-vault"),
							},
							KeyURL: to.Ptr("https://10.0.0.3/keys/my-key/version1"),
						},
					},
				},
			},
		},
	}
}

// createAzureSnapshotWithCrossResourceGroupLinks creates a mock Snapshot with links to resources in different resource groups
func createAzureSnapshotWithCrossResourceGroupLinks(name, subscriptionID string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			DiskAccessID:      to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Compute/diskAccesses/test-disk-access"),
			CreationData: &armcompute.CreationData{
				CreateOption:    to.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/disk-rg/providers/Microsoft.Compute/disks/source-disk"),
			},
		},
	}
}

// createAzureSnapshotWithoutLinks creates a mock Snapshot without any linked resources
func createAzureSnapshotWithoutLinks(name string) *armcompute.Snapshot {
	return &armcompute.Snapshot{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.SnapshotProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionEmpty),
			},
		},
	}
}

// mockSnapshotsPager is a simple mock implementation of the Pager interface for testing
type mockSnapshotsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.Snapshot
	index int
	more  bool
}

func newMockSnapshotsPager(ctrl *gomock.Controller, items []*armcompute.Snapshot) clients.SnapshotsPager {
	return &mockSnapshotsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockSnapshotsPager) More() bool {
	return m.more
}

func (m *mockSnapshotsPager) NextPage(ctx context.Context) (armcompute.SnapshotsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.SnapshotsClientListByResourceGroupResponse{
			SnapshotList: armcompute.SnapshotList{
				Value: []*armcompute.Snapshot{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.SnapshotsClientListByResourceGroupResponse{
		SnapshotList: armcompute.SnapshotList{
			Value: []*armcompute.Snapshot{item},
		},
	}, nil
}
