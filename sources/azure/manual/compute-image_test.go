package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		imageName := "test-image"
		image := createAzureImage(imageName)

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, imageName, nil).Return(
			armcompute.ImagesClientGetResponse{
				Image: *image,
			}, nil)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], imageName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeImage.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeImage, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != imageName {
			t.Errorf("Expected unique attribute value %s, got %s", imageName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}
	})

	t.Run("GetWithAllLinkedResources", func(t *testing.T) {
		imageName := "test-image"
		image := createAzureImageWithAllLinks(imageName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, imageName, nil).Return(
			armcompute.ImagesClientGetResponse{
				Image: *image,
			}, nil)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], imageName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// OSDisk.ManagedDisk.ID - Compute Disk
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-os-disk",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// OSDisk.Snapshot.ID - Compute Snapshot
					ExpectedType:   azureshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-os-snapshot",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// OSDisk.BlobURI - Storage Account
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "teststorageaccount",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// OSDisk.BlobURI - NetworkHTTP
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://teststorageaccount.blob.core.windows.net/vhds/osdisk.vhd",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// OSDisk.BlobURI - NetworkDNS
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "teststorageaccount.blob.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// OSDisk.DiskEncryptionSet.ID - Disk Encryption Set
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-os-disk-encryption-set",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DataDisks[0].ManagedDisk.ID - Compute Disk
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-data-disk-1",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DataDisks[0].Snapshot.ID - Compute Snapshot
					ExpectedType:   azureshared.ComputeSnapshot.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-data-snapshot-1",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DataDisks[0].BlobURI - Storage Account
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "teststorageaccount2",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DataDisks[0].BlobURI - NetworkHTTP
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://teststorageaccount2.blob.core.windows.net/vhds/datadisk1.vhd",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DataDisks[0].BlobURI - NetworkDNS
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "teststorageaccount2.blob.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DataDisks[0].DiskEncryptionSet.ID - Disk Encryption Set
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-data-disk-encryption-set",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// SourceVirtualMachine.ID - Virtual Machine
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-source-vm",
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

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		imageName := "test-image-cross-rg"
		image := createAzureImageWithCrossResourceGroupLinks(imageName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, imageName, nil).Return(
			armcompute.ImagesClientGetResponse{
				Image: *image,
			}, nil)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], imageName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that links to resources in different resource groups use the correct scope
		foundCrossRGLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.ComputeDisk.String() &&
				linkedQuery.GetQuery().GetQuery() == "test-disk-other-rg" {
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
		image1 := createAzureImage("test-image-1")
		image2 := createAzureImage("test-image-2")

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockPager := newMockImagesPager(ctrl, []*armcompute.Image{image1, image2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
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
		image1 := createAzureImage("test-image-1")
		image2 := createAzureImage("test-image-2")

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockPager := newMockImagesPager(ctrl, []*armcompute.Image{image1, image2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		image1 := createAzureImage("test-image-1")
		imageNilName := &armcompute.Image{
			Name:     nil, // nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockPager := newMockImagesPager(ctrl, []*armcompute.Image{image1, imageNilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		expectedErr := errors.New("image not found")

		mockClient := mocks.NewMockImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-image", nil).Return(
			armcompute.ImagesClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-image", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent image, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		// Test the wrapper's Get method directly with insufficient query parts
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when getting image with insufficient query parts, but got nil")
		}
	})

	t.Run("ListWithPagerError", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		errorPager := newErrorImagesPager(ctrl)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		mockClient := mocks.NewMockImagesClient(ctrl)
		errorPager := newErrorImagesPager(ctrl)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)

		lookups := wrapper.GetLookups()
		if len(lookups) != 1 {
			t.Errorf("Expected 1 lookup, got %d", len(lookups))
		}

		// Verify the lookup is for name
		if lookups[0].By != "name" {
			t.Errorf("Expected lookup attribute 'name', got %s", lookups[0].By)
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)

		potentialLinks := wrapper.PotentialLinks()
		expectedLinks := []shared.ItemType{
			azureshared.ComputeDisk,
			azureshared.ComputeSnapshot,
			azureshared.ComputeDiskEncryptionSet,
			azureshared.ComputeVirtualMachine,
			azureshared.StorageAccount,
			stdlib.NetworkHTTP,
			stdlib.NetworkDNS,
		}

		for _, expectedLink := range expectedLinks {
			if !potentialLinks[expectedLink] {
				t.Errorf("Expected potential link %s to be true", expectedLink)
			}
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)

		mappings := wrapper.TerraformMappings()
		if len(mappings) != 1 {
			t.Errorf("Expected 1 terraform mapping, got %d", len(mappings))
		}

		if mappings[0].GetTerraformMethod() != sdp.QueryMethod_GET {
			t.Errorf("Expected terraform method GET, got %v", mappings[0].GetTerraformMethod())
		}

		if mappings[0].GetTerraformQueryMap() != "azurerm_image.name" {
			t.Errorf("Expected terraform query map 'azurerm_image.name', got %s", mappings[0].GetTerraformQueryMap())
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)

		permissions := wrapper.IAMPermissions()
		expectedPermissions := []string{
			"Microsoft.Compute/images/read",
		}

		if len(permissions) != len(expectedPermissions) {
			t.Errorf("Expected %d permissions, got %d", len(expectedPermissions), len(permissions))
		}

		for i, expected := range expectedPermissions {
			if permissions[i] != expected {
				t.Errorf("Expected permission %s, got %s", expected, permissions[i])
			}
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockImagesClient(ctrl)
		wrapper := manual.NewComputeImage(mockClient, subscriptionID, resourceGroup)

		// PredefinedRole is available on the wrapper, not the adapter
		if roleInterface, ok := interface{}(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected predefined role 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}
	})
}

// createAzureImage creates a mock Azure Image for testing
func createAzureImage(imageName string) *armcompute.Image {
	return &armcompute.Image{
		Name:     to.Ptr(imageName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.ImageProperties{
			ProvisioningState: to.Ptr("Succeeded"),
		},
	}
}

// createAzureImageWithAllLinks creates a mock Azure Image with all possible linked resources
func createAzureImageWithAllLinks(imageName, subscriptionID, resourceGroup string) *armcompute.Image {
	osDiskBlobURI := "https://teststorageaccount.blob.core.windows.net/vhds/osdisk.vhd"
	dataDiskBlobURI := "https://teststorageaccount2.blob.core.windows.net/vhds/datadisk1.vhd"

	return &armcompute.Image{
		Name:     to.Ptr(imageName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.ImageProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			StorageProfile: &armcompute.ImageStorageProfile{
				OSDisk: &armcompute.ImageOSDisk{
					OSType:  to.Ptr(armcompute.OperatingSystemTypesLinux),
					OSState: to.Ptr(armcompute.OperatingSystemStateTypesGeneralized),
					ManagedDisk: &armcompute.SubResource{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/test-os-disk"),
					},
					Snapshot: &armcompute.SubResource{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/snapshots/test-os-snapshot"),
					},
					BlobURI: to.Ptr(osDiskBlobURI),
					DiskEncryptionSet: &armcompute.DiskEncryptionSetParameters{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-os-disk-encryption-set"),
					},
				},
				DataDisks: []*armcompute.ImageDataDisk{
					{
						Lun: to.Ptr(int32(0)),
						ManagedDisk: &armcompute.SubResource{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/test-data-disk-1"),
						},
						Snapshot: &armcompute.SubResource{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/snapshots/test-data-snapshot-1"),
						},
						BlobURI: to.Ptr(dataDiskBlobURI),
						DiskEncryptionSet: &armcompute.DiskEncryptionSetParameters{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-data-disk-encryption-set"),
						},
					},
				},
			},
			SourceVirtualMachine: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/test-source-vm"),
			},
		},
	}
}

// createAzureImageWithCrossResourceGroupLinks creates a mock Azure Image with links to resources in different resource groups
func createAzureImageWithCrossResourceGroupLinks(imageName, subscriptionID, resourceGroup string) *armcompute.Image {
	return &armcompute.Image{
		Name:     to.Ptr(imageName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.ImageProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			StorageProfile: &armcompute.ImageStorageProfile{
				OSDisk: &armcompute.ImageOSDisk{
					OSType:  to.Ptr(armcompute.OperatingSystemTypesLinux),
					OSState: to.Ptr(armcompute.OperatingSystemStateTypesGeneralized),
					ManagedDisk: &armcompute.SubResource{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Compute/disks/test-disk-other-rg"),
					},
				},
			},
		},
	}
}

// mockImagesPager is a simple mock implementation of the Pager interface for testing
type mockImagesPager struct {
	items []*armcompute.Image
	index int
	more  bool
}

func newMockImagesPager(ctrl *gomock.Controller, items []*armcompute.Image) clients.ImagesPager {
	return &mockImagesPager{
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockImagesPager) More() bool {
	return m.more
}

func (m *mockImagesPager) NextPage(ctx context.Context) (armcompute.ImagesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.ImagesClientListByResourceGroupResponse{
			ImageListResult: armcompute.ImageListResult{
				Value: []*armcompute.Image{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.ImagesClientListByResourceGroupResponse{
		ImageListResult: armcompute.ImageListResult{
			Value: []*armcompute.Image{item},
		},
	}, nil
}

// errorImagesPager is a mock pager that always returns an error
type errorImagesPager struct {
}

func newErrorImagesPager(ctrl *gomock.Controller) clients.ImagesPager {
	return &errorImagesPager{}
}

func (e *errorImagesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorImagesPager) NextPage(ctx context.Context) (armcompute.ImagesClientListByResourceGroupResponse, error) {
	return armcompute.ImagesClientListByResourceGroupResponse{}, errors.New("pager error")
}
