package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestCapacityReservationGroupName = "ovm-integ-test-capacity-reservation-group"
)

func TestComputeCapacityReservationGroupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	capacityReservationGroupsClient, err := armcompute.NewCapacityReservationGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Capacity Reservation Groups client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createCapacityReservationGroup(ctx, capacityReservationGroupsClient, integrationTestResourceGroup, integrationTestCapacityReservationGroupName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create capacity reservation group: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetCapacityReservationGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving capacity reservation group %s in subscription %s, resource group %s",
				integrationTestCapacityReservationGroupName, subscriptionID, integrationTestResourceGroup)

			capacityReservationGroupWrapper := manual.NewComputeCapacityReservationGroup(
				clients.NewCapacityReservationGroupsClient(capacityReservationGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := capacityReservationGroupWrapper.Scopes()[0]

			capacityReservationGroupAdapter := sources.WrapperToAdapter(capacityReservationGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := capacityReservationGroupAdapter.Get(ctx, scope, integrationTestCapacityReservationGroupName, true)
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

			if uniqueAttrValue != integrationTestCapacityReservationGroupName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestCapacityReservationGroupName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved capacity reservation group %s", integrationTestCapacityReservationGroupName)
		})

		t.Run("ListCapacityReservationGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing capacity reservation groups in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			capacityReservationGroupWrapper := manual.NewComputeCapacityReservationGroup(
				clients.NewCapacityReservationGroupsClient(capacityReservationGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := capacityReservationGroupWrapper.Scopes()[0]

			capacityReservationGroupAdapter := sources.WrapperToAdapter(capacityReservationGroupWrapper, sdpcache.NewNoOpCache())

			listable, ok := capacityReservationGroupAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list capacity reservation groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one capacity reservation group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestCapacityReservationGroupName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find capacity reservation group %s in the list of capacity reservation groups", integrationTestCapacityReservationGroupName)
			}

			log.Printf("Found %d capacity reservation groups in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for capacity reservation group %s", integrationTestCapacityReservationGroupName)

			capacityReservationGroupWrapper := manual.NewComputeCapacityReservationGroup(
				clients.NewCapacityReservationGroupsClient(capacityReservationGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := capacityReservationGroupWrapper.Scopes()[0]

			capacityReservationGroupAdapter := sources.WrapperToAdapter(capacityReservationGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := capacityReservationGroupAdapter.Get(ctx, scope, integrationTestCapacityReservationGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeCapacityReservationGroup.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeCapacityReservationGroup.String(), sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for capacity reservation group %s", integrationTestCapacityReservationGroupName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for capacity reservation group %s", integrationTestCapacityReservationGroupName)

			capacityReservationGroupWrapper := manual.NewComputeCapacityReservationGroup(
				clients.NewCapacityReservationGroupsClient(capacityReservationGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := capacityReservationGroupWrapper.Scopes()[0]

			capacityReservationGroupAdapter := sources.WrapperToAdapter(capacityReservationGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := capacityReservationGroupAdapter.Get(ctx, scope, integrationTestCapacityReservationGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for capacity reservation group %s", len(linkedQueries), integrationTestCapacityReservationGroupName)

			// Capacity reservation group may have zero or more linked queries (capacity reservations, VMs) depending on configuration
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteCapacityReservationGroup(ctx, capacityReservationGroupsClient, integrationTestResourceGroup, integrationTestCapacityReservationGroupName)
		if err != nil {
			t.Fatalf("Failed to delete capacity reservation group: %v", err)
		}
	})
}

// createCapacityReservationGroup creates an Azure capacity reservation group resource (idempotent).
func createCapacityReservationGroup(ctx context.Context, client *armcompute.CapacityReservationGroupsClient, resourceGroupName, groupName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, groupName, nil)
	if err == nil {
		log.Printf("Capacity reservation group %s already exists, skipping creation", groupName)
		return nil
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected error checking capacity reservation group: %w", err)
	}

	_, err = client.CreateOrUpdate(ctx, resourceGroupName, groupName, armcompute.CapacityReservationGroup{
		Location: new(location),
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("compute-capacity-reservation-group"),
		},
	}, nil)
	if err != nil {
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Capacity reservation group %s already exists (conflict), skipping creation", groupName)
			return nil
		}
		return fmt.Errorf("failed to create capacity reservation group: %w", err)
	}

	log.Printf("Capacity reservation group %s created successfully", groupName)
	return nil
}

// deleteCapacityReservationGroup deletes an Azure capacity reservation group resource.
// Azure may return 202 Accepted for async delete; treat that as success.
func deleteCapacityReservationGroup(ctx context.Context, client *armcompute.CapacityReservationGroupsClient, resourceGroupName, groupName string) error {
	_, err := client.Delete(ctx, resourceGroupName, groupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			switch respErr.StatusCode {
			case http.StatusNotFound:
				log.Printf("Capacity reservation group %s not found, skipping deletion", groupName)
				return nil
			case http.StatusAccepted:
				// Async delete accepted; resource deletion is in progress
				log.Printf("Capacity reservation group %s delete accepted (202), teardown complete", groupName)
				return nil
			}
		}
		return fmt.Errorf("failed to delete capacity reservation group: %w", err)
	}

	log.Printf("Capacity reservation group %s deleted successfully", groupName)
	return nil
}
