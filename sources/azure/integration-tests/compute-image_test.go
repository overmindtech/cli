package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestImageName     = "ovm-integ-test-image"
	integrationTestImageDiskName = "ovm-integ-test-image-disk"
)

func TestComputeImageIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	// Initialize Azure credentials using DefaultAzureCredential
	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	// Create Azure SDK clients
	imageClient, err := armcompute.NewImagesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Images client: %v", err)
	}

	diskClient, err := armcompute.NewDisksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Disks client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create disk first (required for image creation)
		err = createDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestImageDiskName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create disk: %v", err)
		}

		// Wait for disk to be fully available
		err = waitForDiskAvailable(ctx, diskClient, integrationTestResourceGroup, integrationTestImageDiskName)
		if err != nil {
			t.Fatalf("Failed waiting for disk to be available: %v", err)
		}

		// Get the disk ID for image creation
		disk, err := diskClient.Get(ctx, integrationTestResourceGroup, integrationTestImageDiskName, nil)
		if err != nil {
			t.Fatalf("Failed to get disk: %v", err)
		}
		if disk.ID == nil || *disk.ID == "" {
			t.Fatalf("Disk ID is nil or empty")
		}

		// Create image from the disk
		err = createImage(ctx, imageClient, integrationTestResourceGroup, integrationTestImageName, integrationTestLocation, *disk.ID)
		if err != nil {
			t.Fatalf("Failed to create image: %v", err)
		}

		// Wait for image to be fully available
		err = waitForImageAvailable(ctx, imageClient, integrationTestResourceGroup, integrationTestImageName)
		if err != nil {
			t.Fatalf("Failed waiting for image to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetImage", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving image %s in subscription %s, resource group %s",
				integrationTestImageName, subscriptionID, integrationTestResourceGroup)

			imageWrapper := manual.NewComputeImage(
				clients.NewImagesClient(imageClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := imageWrapper.Scopes()[0]

			imageAdapter := sources.WrapperToAdapter(imageWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := imageAdapter.Get(ctx, scope, integrationTestImageName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != integrationTestImageName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestImageName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved image %s", integrationTestImageName)
		})

		t.Run("ListImages", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing images in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			imageWrapper := manual.NewComputeImage(
				clients.NewImagesClient(imageClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := imageWrapper.Scopes()[0]

			imageAdapter := sources.WrapperToAdapter(imageWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := imageAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list images: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one image, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestImageName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find image %s in the list of images", integrationTestImageName)
			}

			log.Printf("Found %d images in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for image %s", integrationTestImageName)

			imageWrapper := manual.NewComputeImage(
				clients.NewImagesClient(imageClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := imageWrapper.Scopes()[0]

			imageAdapter := sources.WrapperToAdapter(imageWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := imageAdapter.Get(ctx, scope, integrationTestImageName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.ComputeImage.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeImage, sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for image %s", integrationTestImageName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for image %s", integrationTestImageName)

			imageWrapper := manual.NewComputeImage(
				clients.NewImagesClient(imageClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := imageWrapper.Scopes()[0]

			imageAdapter := sources.WrapperToAdapter(imageWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := imageAdapter.Get(ctx, scope, integrationTestImageName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (image should link to the source disk)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for image %s", len(linkedQueries), integrationTestImageName)

			// An image created from a managed disk should have at least one linked item (the disk)
			if len(linkedQueries) < 1 {
				t.Errorf("Expected at least one linked item query for image created from disk, got %d", len(linkedQueries))
			}

			// Verify linked item structure
			var foundDiskLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				// Verify query has required fields
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				// Method should be GET or SEARCH (not empty)
				if query.GetMethod() == sdp.QueryMethod_GET || query.GetMethod() == sdp.QueryMethod_SEARCH {
					// Valid method
				} else {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				// Verify blast propagation is set
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
					continue
				}

				// Blast propagation should have In and Out set (even if false)
				_ = bp.GetIn()
				_ = bp.GetOut()

				// Check if this is a link to the source disk
				if query.GetType() == azureshared.ComputeDisk.String() && query.GetQuery() == integrationTestImageDiskName {
					foundDiskLink = true
					// Verify blast propagation for disk link
					if !bp.GetIn() {
						t.Error("Expected In=true for disk link (if source disk is deleted/modified, image becomes invalid)")
					}
					if bp.GetOut() {
						t.Error("Expected Out=false for disk link (if image is deleted, source disk remains)")
					}
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s, In=%v, Out=%v",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope(),
					bp.GetIn(), bp.GetOut())
			}

			// Verify we found the expected disk link
			if !foundDiskLink {
				t.Errorf("Expected to find linked item query for disk %s, but it was not found", integrationTestImageDiskName)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete image first
		err := deleteImage(ctx, imageClient, integrationTestResourceGroup, integrationTestImageName)
		if err != nil {
			t.Fatalf("Failed to delete image: %v", err)
		}

		// Delete disk
		err = deleteDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestImageDiskName)
		if err != nil {
			t.Fatalf("Failed to delete disk: %v", err)
		}

		// Optionally delete the resource group
		// Note: We keep the resource group for faster subsequent test runs
		// Uncomment the following if you want to clean up completely:
		// err = deleteResourceGroup(ctx, rgClient, integrationTestResourceGroup)
		// if err != nil {
		//     t.Fatalf("Failed to delete resource group: %v", err)
		// }
	})
}

// createImage creates an Azure compute image from a managed disk (idempotent)
func createImage(ctx context.Context, client *armcompute.ImagesClient, resourceGroupName, imageName, location, sourceDiskID string) error {
	// Check if image already exists
	existingImage, err := client.Get(ctx, resourceGroupName, imageName, nil)
	if err == nil {
		// Image exists, check its state
		if existingImage.Properties != nil && existingImage.Properties.ProvisioningState != nil {
			state := *existingImage.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Image %s already exists with state %s, skipping creation", imageName, state)
				return nil
			}
			log.Printf("Image %s exists but in state %s, will wait for it", imageName, state)
		} else {
			log.Printf("Image %s already exists, skipping creation", imageName)
			return nil
		}
	}

	// Create an image from a managed disk
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, imageName, armcompute.Image{
		Location: ptr.To(location),
		Properties: &armcompute.ImageProperties{
			HyperVGeneration: ptr.To(armcompute.HyperVGenerationTypesV1),
			StorageProfile: &armcompute.ImageStorageProfile{
				OSDisk: &armcompute.ImageOSDisk{
					ManagedDisk: &armcompute.SubResource{
						ID: ptr.To(sourceDiskID),
					},
					OSState: ptr.To(armcompute.OperatingSystemStateTypesGeneralized),
					OSType:  ptr.To(armcompute.OperatingSystemTypesLinux),
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-image"),
		},
	}, nil)
	if err != nil {
		// Check if image already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Image %s already exists (conflict), skipping creation", imageName)
			return nil
		}
		return fmt.Errorf("failed to begin creating image: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	// Verify the image was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("image created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("image provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Image %s created successfully with provisioning state: %s", imageName, provisioningState)
	return nil
}

// waitForImageAvailable polls until the image is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the image is queryable
func waitForImageAvailable(ctx context.Context, client *armcompute.ImagesClient, resourceGroupName, imageName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for image %s to be available via API...", imageName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, imageName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Image %s not yet available (attempt %d/%d), waiting %v...", imageName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking image availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Image %s is available with provisioning state: %s", imageName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("image provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Image %s provisioning state: %s (attempt %d/%d), waiting...", imageName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Image exists but no provisioning state - consider it available
		log.Printf("Image %s is available", imageName)
		return nil
	}

	return fmt.Errorf("timeout waiting for image %s to be available after %d attempts", imageName, maxAttempts)
}

// deleteImage deletes an Azure compute image
func deleteImage(ctx context.Context, client *armcompute.ImagesClient, resourceGroupName, imageName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, imageName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Image %s not found, skipping deletion", imageName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting image: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	log.Printf("Image %s deleted successfully", imageName)
	return nil
}
