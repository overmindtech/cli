package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeDisk(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		diskName := "test-disk"
		disk := createAzureDisk(diskName, "Succeeded")

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskName, nil).Return(
			armcompute.DisksClientGetResponse{
				Disk: *disk,
			}, nil)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], diskName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDisk.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDisk, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != diskName {
			t.Errorf("Expected unique attribute value %s, got %s", diskName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}
	})

	t.Run("GetWithAllLinkedResources", func(t *testing.T) {
		diskName := "test-disk"
		disk := createAzureDiskWithAllLinks(diskName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskName, nil).Return(
			armcompute.DisksClientGetResponse{
				Disk: *disk,
			}, nil)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], diskName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// ManagedBy - Virtual Machine
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// ManagedByExtended[0] - Virtual Machine
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm-2",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// ShareInfo[0].VMURI - Virtual Machine
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm-3",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.DiskAccessID - Disk Access
					ExpectedType:   azureshared.ComputeDiskAccess.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk-access",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.Encryption.DiskEncryptionSetID - Disk Encryption Set
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk-encryption-set",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.SecurityProfile.SecureVMDiskEncryptionSetID - Disk Encryption Set
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-secure-vm-disk-encryption-set",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.SourceResourceID (Disk) - Source Disk
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "source-disk",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.StorageAccountID - Storage Account
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-storage-account",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.ImageReference.ID - Image
					ExpectedType:   azureshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-image",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.GalleryImageReference.ID - Shared Gallery Image
					ExpectedType:   azureshared.ComputeSharedGalleryImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-gallery", "test-gallery-image", "1.0.0"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.GalleryImageReference.SharedGalleryImageID - Shared Gallery Image
					ExpectedType:   azureshared.ComputeSharedGalleryImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-gallery-2", "test-gallery-image-2", "2.0.0"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.GalleryImageReference.CommunityGalleryImageID - Community Gallery Image
					ExpectedType:   azureshared.ComputeCommunityGalleryImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-community-gallery", "test-community-image", "1.0.0"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.CreationData.ElasticSanResourceID - Elastic SAN Volume Snapshot
					ExpectedType:   azureshared.ElasticSanVolumeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-elastic-san", "test-volume-group", "test-snapshot"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.EncryptionSettingsCollection.EncryptionSettings[0].DiskEncryptionKey.SourceVault.ID - Key Vault
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-keyvault",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.EncryptionSettingsCollection.EncryptionSettings[0].DiskEncryptionKey.SecretURL - Key Vault Secret
					ExpectedType:   azureshared.KeyVaultSecret.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-keyvault", "test-secret"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.EncryptionSettingsCollection.EncryptionSettings[0].KeyEncryptionKey.SourceVault.ID - Key Vault
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-keyvault-2",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.EncryptionSettingsCollection.EncryptionSettings[0].KeyEncryptionKey.KeyURL - Key Vault Key
					ExpectedType:   azureshared.KeyVaultKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-keyvault-2", "test-key"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithSnapshotSource", func(t *testing.T) {
		diskName := "test-disk-from-snapshot"
		disk := createAzureDiskFromSnapshot(diskName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskName, nil).Return(
			armcompute.DisksClientGetResponse{
				Disk: *disk,
			}, nil)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], diskName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify snapshot link
		foundSnapshotLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.ComputeSnapshot.String() &&
				linkedQuery.GetQuery().GetQuery() == "test-snapshot" {
				foundSnapshotLink = true
				if linkedQuery.GetBlastPropagation().GetIn() != true {
					t.Errorf("Expected BlastPropagation.In to be true for snapshot link")
				}
				if linkedQuery.GetBlastPropagation().GetOut() != false {
					t.Errorf("Expected BlastPropagation.Out to be false for snapshot link")
				}
				break
			}
		}
		if !foundSnapshotLink {
			t.Error("Expected snapshot link not found")
		}
	})

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		diskName := "test-disk-cross-rg"
		disk := createAzureDiskWithCrossResourceGroupLinks(diskName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskName, nil).Return(
			armcompute.DisksClientGetResponse{
				Disk: *disk,
			}, nil)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], diskName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that links to resources in different resource groups use the correct scope
		foundCrossRGLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.ComputeVirtualMachine.String() &&
				linkedQuery.GetQuery().GetQuery() == "test-vm-other-rg" {
				foundCrossRGLink = true
				expectedScope := subscriptionID + ".other-rg"
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected scope %s for cross-RG link, got %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				break
			}
		}
		if !foundCrossRGLink {
			t.Error("Expected cross-resource-group link not found")
		}
	})

	t.Run("List", func(t *testing.T) {
		disk1 := createAzureDisk("test-disk-1", "Succeeded")
		disk2 := createAzureDisk("test-disk-2", "Succeeded")

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockPager := newMockDisksPager(ctrl, []*armcompute.Disk{disk1, disk2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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
		disk1 := createAzureDisk("test-disk-1", "Succeeded")
		disk2 := createAzureDisk("test-disk-2", "Succeeded")

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockPager := newMockDisksPager(ctrl, []*armcompute.Disk{disk1, disk2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		disk1 := createAzureDisk("test-disk-1", "Succeeded")
		diskNilName := &armcompute.Disk{
			Name:     nil, // nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockPager := newMockDisksPager(ctrl, []*armcompute.Disk{disk1, diskNilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("disk not found")

		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-disk", nil).Return(
			armcompute.DisksClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-disk", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent disk, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		expectedErr := errors.New("disk name cannot be empty")
		mockClient := mocks.NewMockDisksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armcompute.DisksClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting disk with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDisksClient(ctrl)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		// Test the wrapper's Get method directly with insufficient query parts
		_, qErr := wrapper.Get(ctx)
		if qErr == nil {
			t.Error("Expected error when getting disk with insufficient query parts, but got nil")
		}
	})

	t.Run("ListWithPagerError", func(t *testing.T) {
		mockClient := mocks.NewMockDisksClient(ctrl)
		errorPager := newErrorDisksPager(ctrl)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("ListStreamWithPagerError", func(t *testing.T) {
		mockClient := mocks.NewMockDisksClient(ctrl)
		errorPager := newErrorDisksPager(ctrl)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDisk(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(func(item *sdp.Item) {}, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)

		if len(errs) == 0 {
			t.Error("Expected error when pager returns error, but got none")
		}
	})
}

// createAzureDisk creates a mock Azure Disk for testing
func createAzureDisk(diskName, provisioningState string) *armcompute.Disk {
	return &armcompute.Disk{
		Name:     to.Ptr(diskName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.DiskProperties{
			ProvisioningState: to.Ptr(provisioningState),
			DiskSizeGB:        to.Ptr(int32(128)),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionEmpty),
			},
		},
	}
}

// createAzureDiskWithAllLinks creates a mock Azure Disk with all possible linked resources
func createAzureDiskWithAllLinks(diskName, subscriptionID, resourceGroup string) *armcompute.Disk {
	return &armcompute.Disk{
		Name:     to.Ptr(diskName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		ManagedBy: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/test-vm"),
		ManagedByExtended: []*string{
			to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/test-vm-2"),
		},
		Properties: &armcompute.DiskProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			DiskSizeGB:        to.Ptr(int32(128)),
			DiskAccessID:      to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskAccesses/test-disk-access"),
			Encryption: &armcompute.Encryption{
				DiskEncryptionSetID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set"),
			},
			SecurityProfile: &armcompute.DiskSecurityProfile{
				SecureVMDiskEncryptionSetID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-secure-vm-disk-encryption-set"),
			},
			ShareInfo: []*armcompute.ShareInfoElement{
				{
					VMURI: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/test-vm-3"),
				},
			},
			CreationData: &armcompute.CreationData{
				CreateOption:     to.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/source-disk"),
				StorageAccountID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/test-storage-account"),
				ImageReference: &armcompute.ImageDiskReference{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/images/test-image"),
				},
				GalleryImageReference: &armcompute.ImageDiskReference{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/galleries/test-gallery/images/test-gallery-image/versions/1.0.0"),
					SharedGalleryImageID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/galleries/test-gallery-2/images/test-gallery-image-2/versions/2.0.0"),
					CommunityGalleryImageID: to.Ptr("/CommunityGalleries/test-community-gallery/Images/test-community-image/Versions/1.0.0"),
				},
				ElasticSanResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ElasticSan/elasticSans/test-elastic-san/volumegroups/test-volume-group/snapshots/test-snapshot"),
			},
			EncryptionSettingsCollection: &armcompute.EncryptionSettingsCollection{
				Enabled: to.Ptr(true),
				EncryptionSettings: []*armcompute.EncryptionSettingsElement{
					{
						DiskEncryptionKey: &armcompute.KeyVaultAndSecretReference{
							SourceVault: &armcompute.SourceVault{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-keyvault"),
							},
							SecretURL: to.Ptr("https://test-keyvault.vault.azure.net/secrets/test-secret/version"),
						},
						KeyEncryptionKey: &armcompute.KeyVaultAndKeyReference{
							SourceVault: &armcompute.SourceVault{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-keyvault-2"),
							},
							KeyURL: to.Ptr("https://test-keyvault-2.vault.azure.net/keys/test-key/version"),
						},
					},
				},
			},
		},
	}
}

// createAzureDiskFromSnapshot creates a mock Azure Disk created from a snapshot
func createAzureDiskFromSnapshot(diskName, subscriptionID, resourceGroup string) *armcompute.Disk {
	return &armcompute.Disk{
		Name:     to.Ptr(diskName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.DiskProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			DiskSizeGB:        to.Ptr(int32(128)),
			CreationData: &armcompute.CreationData{
				CreateOption:     to.Ptr(armcompute.DiskCreateOptionCopy),
				SourceResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/snapshots/test-snapshot"),
			},
		},
	}
}

// createAzureDiskWithCrossResourceGroupLinks creates a mock Azure Disk with links to resources in different resource groups
func createAzureDiskWithCrossResourceGroupLinks(diskName, subscriptionID, resourceGroup string) *armcompute.Disk {
	return &armcompute.Disk{
		Name:     to.Ptr(diskName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		ManagedBy: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Compute/virtualMachines/test-vm-other-rg"),
		Properties: &armcompute.DiskProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			DiskSizeGB:        to.Ptr(int32(128)),
			CreationData: &armcompute.CreationData{
				CreateOption: to.Ptr(armcompute.DiskCreateOptionEmpty),
			},
		},
	}
}

// mockDisksPager is a simple mock implementation of the Pager interface for testing
type mockDisksPager struct {
	ctrl     *gomock.Controller
	items    []*armcompute.Disk
	index    int
	more     bool
}

func newMockDisksPager(ctrl *gomock.Controller, items []*armcompute.Disk) clients.DisksPager {
	return &mockDisksPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockDisksPager) More() bool {
	return m.more
}

func (m *mockDisksPager) NextPage(ctx context.Context) (armcompute.DisksClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.DisksClientListByResourceGroupResponse{
			DiskList: armcompute.DiskList{
				Value: []*armcompute.Disk{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.DisksClientListByResourceGroupResponse{
		DiskList: armcompute.DiskList{
			Value: []*armcompute.Disk{item},
		},
	}, nil
}

// errorDisksPager is a mock pager that always returns an error
type errorDisksPager struct {
	ctrl *gomock.Controller
}

func newErrorDisksPager(ctrl *gomock.Controller) clients.DisksPager {
	return &errorDisksPager{ctrl: ctrl}
}

func (e *errorDisksPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorDisksPager) NextPage(ctx context.Context) (armcompute.DisksClientListByResourceGroupResponse, error) {
	return armcompute.DisksClientListByResourceGroupResponse{}, errors.New("pager error")
}

