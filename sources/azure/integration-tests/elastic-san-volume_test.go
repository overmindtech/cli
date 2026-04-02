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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestElasticSanName    = "ovm-integ-test-esan"
	integrationTestVolumeGroupName   = "ovm-integ-test-vg"
	integrationTestVolumeName        = "ovm-integ-test-vol"
	integrationTestElasticSanBaseTiB = int64(1)
	integrationTestVolumeSizeGiB     = int64(1)
)

func TestElasticSanVolumeIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	// Create Azure SDK clients
	esClient, err := armelasticsan.NewElasticSansClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Elastic SAN client: %v", err)
	}

	vgClient, err := armelasticsan.NewVolumeGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Volume Groups client: %v", err)
	}

	volClient, err := armelasticsan.NewVolumesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Volumes client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create Elastic SAN
		err = createElasticSan(ctx, esClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestLocation, integrationTestElasticSanBaseTiB)
		if err != nil {
			t.Fatalf("Failed to create Elastic SAN: %v", err)
		}

		// Wait for Elastic SAN to be available
		err = waitForElasticSanAvailable(ctx, esClient, integrationTestResourceGroup, integrationTestElasticSanName)
		if err != nil {
			t.Fatalf("Failed waiting for Elastic SAN to be available: %v", err)
		}

		// Create Volume Group
		err = createVolumeGroup(ctx, vgClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName)
		if err != nil {
			t.Fatalf("Failed to create Volume Group: %v", err)
		}

		// Wait for Volume Group to be available
		err = waitForVolumeGroupAvailable(ctx, vgClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName)
		if err != nil {
			t.Fatalf("Failed waiting for Volume Group to be available: %v", err)
		}

		// Create Volume
		err = createVolume(ctx, volClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName, integrationTestVolumeSizeGiB)
		if err != nil {
			t.Fatalf("Failed to create Volume: %v", err)
		}

		// Wait for Volume to be available
		err = waitForVolumeAvailable(ctx, volClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
		if err != nil {
			t.Fatalf("Failed waiting for Volume to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetVolume", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving volume %s in volume group %s, elastic san %s, subscription %s, resource group %s",
				integrationTestVolumeName, integrationTestVolumeGroupName, integrationTestElasticSanName, subscriptionID, integrationTestResourceGroup)

			volWrapper := manual.NewElasticSanVolume(
				clients.NewElasticSanVolumeClient(volClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := volWrapper.Scopes()[0]

			volAdapter := sources.WrapperToAdapter(volWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
			sdpItem, qErr := volAdapter.Get(ctx, scope, query, true)
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

			expectedUnique := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
			if uniqueAttrValue != expectedUnique {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved volume %s", integrationTestVolumeName)
		})

		t.Run("SearchVolumes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching volumes in volume group %s, elastic san %s", integrationTestVolumeGroupName, integrationTestElasticSanName)

			volWrapper := manual.NewElasticSanVolume(
				clients.NewElasticSanVolumeClient(volClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := volWrapper.Scopes()[0]

			volAdapter := sources.WrapperToAdapter(volWrapper, sdpcache.NewNoOpCache())

			searchable, ok := volAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			query := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName)
			sdpItems, err := searchable.Search(ctx, scope, query, true)
			if err != nil {
				t.Fatalf("Failed to search volumes: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one volume, got %d", len(sdpItems))
			}

			var found bool
			expectedUnique := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUnique {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find volume %s in the search results", integrationTestVolumeName)
			}

			log.Printf("Found %d volumes in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for volume %s", integrationTestVolumeName)

			volWrapper := manual.NewElasticSanVolume(
				clients.NewElasticSanVolumeClient(volClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := volWrapper.Scopes()[0]

			volAdapter := sources.WrapperToAdapter(volWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
			sdpItem, qErr := volAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasElasticSanLink bool
			var hasVolumeGroupLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				if query.GetType() == azureshared.ElasticSan.String() {
					hasElasticSanLink = true
					if query.GetQuery() != integrationTestElasticSanName {
						t.Errorf("Expected linked query to elastic san %s, got %s", integrationTestElasticSanName, query.GetQuery())
					}
				}
				if query.GetType() == azureshared.ElasticSanVolumeGroup.String() {
					hasVolumeGroupLink = true
					expectedQuery := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName)
					if query.GetQuery() != expectedQuery {
						t.Errorf("Expected linked query to volume group %s, got %s", expectedQuery, query.GetQuery())
					}
				}
			}

			if !hasElasticSanLink {
				t.Error("Expected linked query to elastic san, but didn't find one")
			}
			if !hasVolumeGroupLink {
				t.Error("Expected linked query to volume group, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for volume %s", len(linkedQueries), integrationTestVolumeName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			volWrapper := manual.NewElasticSanVolume(
				clients.NewElasticSanVolumeClient(volClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := volWrapper.Scopes()[0]

			volAdapter := sources.WrapperToAdapter(volWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
			sdpItem, qErr := volAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.ElasticSanVolume.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolume.String(), sdpItem.GetType())
			}

			// Verify scope
			expectedScope := subscriptionID + "." + integrationTestResourceGroup
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			// Validate item
			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for volume %s", integrationTestVolumeName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete Volume
		err := deleteVolume(ctx, volClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName, integrationTestVolumeName)
		if err != nil {
			t.Logf("Failed to delete volume: %v", err)
		}

		// Delete Volume Group
		err = deleteVolumeGroup(ctx, vgClient, integrationTestResourceGroup, integrationTestElasticSanName, integrationTestVolumeGroupName)
		if err != nil {
			t.Logf("Failed to delete volume group: %v", err)
		}

		// Delete Elastic SAN
		err = deleteElasticSan(ctx, esClient, integrationTestResourceGroup, integrationTestElasticSanName)
		if err != nil {
			t.Logf("Failed to delete elastic san: %v", err)
		}

		// Resource group is kept for faster subsequent test runs
	})
}

// createElasticSan creates an Azure Elastic SAN (idempotent)
func createElasticSan(ctx context.Context, client *armelasticsan.ElasticSansClient, resourceGroupName, elasticSanName, location string, baseSizeTiB int64) error {
	_, err := client.Get(ctx, resourceGroupName, elasticSanName, nil)
	if err == nil {
		log.Printf("Elastic SAN %s already exists, skipping creation", elasticSanName)
		return nil
	}

	extendedCapacitySizeTiB := int64(0)
	poller, err := client.BeginCreate(ctx, resourceGroupName, elasticSanName, armelasticsan.ElasticSan{
		Location: &location,
		Properties: &armelasticsan.Properties{
			BaseSizeTiB:             &baseSizeTiB,
			ExtendedCapacitySizeTiB: &extendedCapacitySizeTiB,
			SKU: &armelasticsan.SKU{
				Name: new(armelasticsan.SKUNamePremiumLRS),
			},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, elasticSanName, nil); getErr == nil {
				log.Printf("Elastic SAN %s already exists (conflict), skipping creation", elasticSanName)
				return nil
			}
			return fmt.Errorf("elastic san %s conflict but not retrievable: %w", elasticSanName, err)
		}
		return fmt.Errorf("failed to create elastic san: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create elastic san: %w", err)
	}

	log.Printf("Elastic SAN %s created successfully", elasticSanName)
	return nil
}

// waitForElasticSanAvailable waits for Elastic SAN to be available
func waitForElasticSanAvailable(ctx context.Context, client *armelasticsan.ElasticSansClient, resourceGroupName, elasticSanName string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, elasticSanName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("elastic san %s not found after %d attempts", elasticSanName, notFoundCount)
				}
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking elastic san: %w", err)
		}
		notFoundCount = 0
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armelasticsan.ProvisioningStatesSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for elastic san %s", elasticSanName)
}

// createVolumeGroup creates an Azure Elastic SAN Volume Group (idempotent)
func createVolumeGroup(ctx context.Context, client *armelasticsan.VolumeGroupsClient, resourceGroupName, elasticSanName, volumeGroupName string) error {
	_, err := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, nil)
	if err == nil {
		log.Printf("Volume Group %s already exists, skipping creation", volumeGroupName)
		return nil
	}

	poller, err := client.BeginCreate(ctx, resourceGroupName, elasticSanName, volumeGroupName, armelasticsan.VolumeGroup{
		Properties: &armelasticsan.VolumeGroupProperties{
			ProtocolType: new(armelasticsan.StorageTargetTypeIscsi),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, nil); getErr == nil {
				log.Printf("Volume Group %s already exists (conflict), skipping creation", volumeGroupName)
				return nil
			}
			return fmt.Errorf("volume group %s conflict but not retrievable: %w", volumeGroupName, err)
		}
		return fmt.Errorf("failed to create volume group: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create volume group: %w", err)
	}

	log.Printf("Volume Group %s created successfully", volumeGroupName)
	return nil
}

// waitForVolumeGroupAvailable waits for Volume Group to be available
func waitForVolumeGroupAvailable(ctx context.Context, client *armelasticsan.VolumeGroupsClient, resourceGroupName, elasticSanName, volumeGroupName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("volume group %s not found after %d attempts", volumeGroupName, notFoundCount)
				}
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking volume group: %w", err)
		}
		notFoundCount = 0
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armelasticsan.ProvisioningStatesSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for volume group %s", volumeGroupName)
}

// createVolume creates an Azure Elastic SAN Volume (idempotent)
func createVolume(ctx context.Context, client *armelasticsan.VolumesClient, resourceGroupName, elasticSanName, volumeGroupName, volumeName string, sizeGiB int64) error {
	_, err := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, nil)
	if err == nil {
		log.Printf("Volume %s already exists, skipping creation", volumeName)
		return nil
	}

	poller, err := client.BeginCreate(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, armelasticsan.Volume{
		Properties: &armelasticsan.VolumeProperties{
			SizeGiB: &sizeGiB,
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, nil); getErr == nil {
				log.Printf("Volume %s already exists (conflict), skipping creation", volumeName)
				return nil
			}
			return fmt.Errorf("volume %s conflict but not retrievable: %w", volumeName, err)
		}
		return fmt.Errorf("failed to create volume: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	log.Printf("Volume %s created successfully", volumeName)
	return nil
}

// waitForVolumeAvailable waits for Volume to be available
func waitForVolumeAvailable(ctx context.Context, client *armelasticsan.VolumesClient, resourceGroupName, elasticSanName, volumeGroupName, volumeName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("volume %s not found after %d attempts", volumeName, notFoundCount)
				}
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking volume: %w", err)
		}
		notFoundCount = 0
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armelasticsan.ProvisioningStatesSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for volume %s", volumeName)
}

// deleteVolume deletes an Azure Elastic SAN Volume
func deleteVolume(ctx context.Context, client *armelasticsan.VolumesClient, resourceGroupName, elasticSanName, volumeGroupName, volumeName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, elasticSanName, volumeGroupName, volumeName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Volume %s not found, skipping deletion", volumeName)
			return nil
		}
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	log.Printf("Volume %s deleted successfully", volumeName)
	return nil
}

// deleteVolumeGroup deletes an Azure Elastic SAN Volume Group
func deleteVolumeGroup(ctx context.Context, client *armelasticsan.VolumeGroupsClient, resourceGroupName, elasticSanName, volumeGroupName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, elasticSanName, volumeGroupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Volume Group %s not found, skipping deletion", volumeGroupName)
			return nil
		}
		return fmt.Errorf("failed to delete volume group: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete volume group: %w", err)
	}

	log.Printf("Volume Group %s deleted successfully", volumeGroupName)
	return nil
}

// deleteElasticSan deletes an Azure Elastic SAN
func deleteElasticSan(ctx context.Context, client *armelasticsan.ElasticSansClient, resourceGroupName, elasticSanName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, elasticSanName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Elastic SAN %s not found, skipping deletion", elasticSanName)
			return nil
		}
		return fmt.Errorf("failed to delete elastic san: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete elastic san: %w", err)
	}

	log.Printf("Elastic SAN %s deleted successfully", elasticSanName)
	return nil
}
