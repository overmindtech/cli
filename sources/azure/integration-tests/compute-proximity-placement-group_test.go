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
	integrationTestProximityPlacementGroupName = "ovm-integ-test-ppg"
)

func TestComputeProximityPlacementGroupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	ppgClient, err := armcompute.NewProximityPlacementGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Proximity Placement Groups client: %v", err)
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

		err = createProximityPlacementGroup(ctx, ppgClient, integrationTestResourceGroup, integrationTestProximityPlacementGroupName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create proximity placement group: %v", err)
		}

		err = waitForProximityPlacementGroupAvailable(ctx, ppgClient, integrationTestResourceGroup, integrationTestProximityPlacementGroupName)
		if err != nil {
			t.Fatalf("Failed waiting for proximity placement group to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		ctx := t.Context()
		_, err := ppgClient.Get(ctx, integrationTestResourceGroup, integrationTestProximityPlacementGroupName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				t.Skipf("Proximity placement group %s does not exist - Setup may have failed. Skipping Run tests.", integrationTestProximityPlacementGroupName)
			}
		}

		t.Run("GetProximityPlacementGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving proximity placement group %s in subscription %s, resource group %s",
				integrationTestProximityPlacementGroupName, subscriptionID, integrationTestResourceGroup)

			ppgWrapper := manual.NewComputeProximityPlacementGroup(
				clients.NewProximityPlacementGroupsClient(ppgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ppgWrapper.Scopes()[0]

			ppgAdapter := sources.WrapperToAdapter(ppgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ppgAdapter.Get(ctx, scope, integrationTestProximityPlacementGroupName, true)
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

			if uniqueAttrValue != integrationTestProximityPlacementGroupName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestProximityPlacementGroupName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.ComputeProximityPlacementGroup.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.ComputeProximityPlacementGroup.String(), sdpItem.GetType())
			}

			log.Printf("Successfully retrieved proximity placement group %s", integrationTestProximityPlacementGroupName)
		})

		t.Run("ListProximityPlacementGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing proximity placement groups in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			ppgWrapper := manual.NewComputeProximityPlacementGroup(
				clients.NewProximityPlacementGroupsClient(ppgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ppgWrapper.Scopes()[0]

			ppgAdapter := sources.WrapperToAdapter(ppgWrapper, sdpcache.NewNoOpCache())

			listable, ok := ppgAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list proximity placement groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one proximity placement group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestProximityPlacementGroupName {
					found = true
					if item.GetType() != azureshared.ComputeProximityPlacementGroup.String() {
						t.Errorf("Expected type %s, got %s", azureshared.ComputeProximityPlacementGroup.String(), item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find proximity placement group %s in the list", integrationTestProximityPlacementGroupName)
			}

			log.Printf("Found %d proximity placement groups in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for proximity placement group %s", integrationTestProximityPlacementGroupName)

			ppgWrapper := manual.NewComputeProximityPlacementGroup(
				clients.NewProximityPlacementGroupsClient(ppgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ppgWrapper.Scopes()[0]

			ppgAdapter := sources.WrapperToAdapter(ppgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ppgAdapter.Get(ctx, scope, integrationTestProximityPlacementGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for proximity placement group %s", len(linkedQueries), integrationTestProximityPlacementGroupName)

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected link method to be GET, got %s", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
				} else {
					// PPG links use In: true, Out: true per adapter
					if !bp.GetIn() || !bp.GetOut() {
						t.Errorf("Expected BlastPropagation In=true Out=true for PPG links, got In=%v Out=%v", bp.GetIn(), bp.GetOut())
					}
				}
			}
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for proximity placement group %s", integrationTestProximityPlacementGroupName)

			ppgWrapper := manual.NewComputeProximityPlacementGroup(
				clients.NewProximityPlacementGroupsClient(ppgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ppgWrapper.Scopes()[0]

			ppgAdapter := sources.WrapperToAdapter(ppgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ppgAdapter.Get(ctx, scope, integrationTestProximityPlacementGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeProximityPlacementGroup.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeProximityPlacementGroup.String(), sdpItem.GetType())
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

			log.Printf("Verified item attributes for proximity placement group %s", integrationTestProximityPlacementGroupName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteProximityPlacementGroup(ctx, ppgClient, integrationTestResourceGroup, integrationTestProximityPlacementGroupName)
		if err != nil {
			t.Fatalf("Failed to delete proximity placement group: %v", err)
		}
	})
}

func createProximityPlacementGroup(ctx context.Context, client *armcompute.ProximityPlacementGroupsClient, resourceGroupName, ppgName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, ppgName, nil)
	if err == nil {
		log.Printf("Proximity placement group %s already exists, skipping creation", ppgName)
		return nil
	}

	resp, err := client.CreateOrUpdate(ctx, resourceGroupName, ppgName, armcompute.ProximityPlacementGroup{
		Location: ptr.To(location),
		Properties: &armcompute.ProximityPlacementGroupProperties{
			ProximityPlacementGroupType: ptr.To(armcompute.ProximityPlacementGroupTypeStandard),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-proximity-placement-group"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Proximity placement group %s already exists (conflict), skipping creation", ppgName)
			return nil
		}
		return fmt.Errorf("failed to create proximity placement group: %w", err)
	}

	if resp.Name == nil {
		return fmt.Errorf("proximity placement group created but name is nil")
	}

	log.Printf("Proximity placement group %s created successfully", ppgName)
	return nil
}

func waitForProximityPlacementGroupAvailable(ctx context.Context, client *armcompute.ProximityPlacementGroupsClient, resourceGroupName, ppgName string) error {
	const maxAttempts = 10
	pollInterval := 2 * time.Second

	log.Printf("Waiting for proximity placement group %s to be available via API...", ppgName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, ppgName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Proximity placement group %s not yet available (attempt %d/%d), waiting %v...", ppgName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking proximity placement group availability: %w", err)
		}

		if resp.Name != nil {
			log.Printf("Proximity placement group %s is available", ppgName)
			return nil
		}

		if attempt < maxAttempts {
			log.Printf("Proximity placement group %s not yet ready (attempt %d/%d), waiting %v...", ppgName, attempt, maxAttempts, pollInterval)
			time.Sleep(pollInterval)
			continue
		}
	}

	return fmt.Errorf("timeout waiting for proximity placement group %s to be available after %d attempts", ppgName, maxAttempts)
}

func deleteProximityPlacementGroup(ctx context.Context, client *armcompute.ProximityPlacementGroupsClient, resourceGroupName, ppgName string) error {
	_, err := client.Delete(ctx, resourceGroupName, ppgName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Proximity placement group %s not found, skipping deletion", ppgName)
			return nil
		}
		return fmt.Errorf("failed to delete proximity placement group: %w", err)
	}

	log.Printf("Proximity placement group %s deleted successfully", ppgName)
	return nil
}
