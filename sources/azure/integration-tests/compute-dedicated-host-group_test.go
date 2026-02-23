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
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestDedicatedHostGroupName = "ovm-integ-test-dedicated-host-group"
)

func TestComputeDedicatedHostGroupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	dedicatedHostGroupsClient, err := armcompute.NewDedicatedHostGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Dedicated Host Groups client: %v", err)
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

		err = createDedicatedHostGroup(ctx, dedicatedHostGroupsClient, integrationTestResourceGroup, integrationTestDedicatedHostGroupName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create dedicated host group: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetDedicatedHostGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving dedicated host group %s in subscription %s, resource group %s",
				integrationTestDedicatedHostGroupName, subscriptionID, integrationTestResourceGroup)

			dedicatedHostGroupWrapper := manual.NewComputeDedicatedHostGroup(
				clients.NewDedicatedHostGroupsClient(dedicatedHostGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := dedicatedHostGroupWrapper.Scopes()[0]

			dedicatedHostGroupAdapter := sources.WrapperToAdapter(dedicatedHostGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := dedicatedHostGroupAdapter.Get(ctx, scope, integrationTestDedicatedHostGroupName, true)
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

			if uniqueAttrValue != integrationTestDedicatedHostGroupName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestDedicatedHostGroupName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved dedicated host group %s", integrationTestDedicatedHostGroupName)
		})

		t.Run("ListDedicatedHostGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing dedicated host groups in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			dedicatedHostGroupWrapper := manual.NewComputeDedicatedHostGroup(
				clients.NewDedicatedHostGroupsClient(dedicatedHostGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := dedicatedHostGroupWrapper.Scopes()[0]

			dedicatedHostGroupAdapter := sources.WrapperToAdapter(dedicatedHostGroupWrapper, sdpcache.NewNoOpCache())

			listable, ok := dedicatedHostGroupAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list dedicated host groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one dedicated host group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestDedicatedHostGroupName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find dedicated host group %s in the list of dedicated host groups", integrationTestDedicatedHostGroupName)
			}

			log.Printf("Found %d dedicated host groups in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for dedicated host group %s", integrationTestDedicatedHostGroupName)

			dedicatedHostGroupWrapper := manual.NewComputeDedicatedHostGroup(
				clients.NewDedicatedHostGroupsClient(dedicatedHostGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := dedicatedHostGroupWrapper.Scopes()[0]

			dedicatedHostGroupAdapter := sources.WrapperToAdapter(dedicatedHostGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := dedicatedHostGroupAdapter.Get(ctx, scope, integrationTestDedicatedHostGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeDedicatedHostGroup.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeDedicatedHostGroup.String(), sdpItem.GetType())
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

			log.Printf("Verified item attributes for dedicated host group %s", integrationTestDedicatedHostGroupName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for dedicated host group %s", integrationTestDedicatedHostGroupName)

			dedicatedHostGroupWrapper := manual.NewComputeDedicatedHostGroup(
				clients.NewDedicatedHostGroupsClient(dedicatedHostGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := dedicatedHostGroupWrapper.Scopes()[0]

			dedicatedHostGroupAdapter := sources.WrapperToAdapter(dedicatedHostGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := dedicatedHostGroupAdapter.Get(ctx, scope, integrationTestDedicatedHostGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for dedicated host group %s", len(linkedQueries), integrationTestDedicatedHostGroupName)

			// Dedicated host group may have zero or more linked queries (ComputeDedicatedHost) depending on whether hosts exist
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

		err := deleteDedicatedHostGroup(ctx, dedicatedHostGroupsClient, integrationTestResourceGroup, integrationTestDedicatedHostGroupName)
		if err != nil {
			t.Fatalf("Failed to delete dedicated host group: %v", err)
		}
	})
}

// createDedicatedHostGroup creates an Azure dedicated host group resource (idempotent).
func createDedicatedHostGroup(ctx context.Context, client *armcompute.DedicatedHostGroupsClient, resourceGroupName, hostGroupName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, hostGroupName, nil)
	if err == nil {
		log.Printf("Dedicated host group %s already exists, skipping creation", hostGroupName)
		return nil
	}

	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected error checking dedicated host group: %w", err)
	}

	_, err = client.CreateOrUpdate(ctx, resourceGroupName, hostGroupName, armcompute.DedicatedHostGroup{
		Location: ptr.To(location),
		Properties: &armcompute.DedicatedHostGroupProperties{
			PlatformFaultDomainCount: ptr.To[int32](1),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-dedicated-host-group"),
		},
	}, nil)
	if err != nil {
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Dedicated host group %s already exists (conflict), skipping creation", hostGroupName)
			return nil
		}
		return fmt.Errorf("failed to create dedicated host group: %w", err)
	}

	log.Printf("Dedicated host group %s created successfully", hostGroupName)
	return nil
}

// deleteDedicatedHostGroup deletes an Azure dedicated host group resource.
func deleteDedicatedHostGroup(ctx context.Context, client *armcompute.DedicatedHostGroupsClient, resourceGroupName, hostGroupName string) error {
	_, err := client.Delete(ctx, resourceGroupName, hostGroupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Dedicated host group %s not found, skipping deletion", hostGroupName)
			return nil
		}
		return fmt.Errorf("failed to delete dedicated host group: %w", err)
	}

	log.Printf("Dedicated host group %s deleted successfully", hostGroupName)
	return nil
}
