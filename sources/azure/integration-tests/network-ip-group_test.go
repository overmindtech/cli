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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
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
	integrationTestIPGroupName = "ovm-integ-test-ip-group"
)

func TestNetworkIPGroupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	ipGroupsClient, err := armnetwork.NewIPGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create IP Groups client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createIPGroup(ctx, ipGroupsClient, integrationTestResourceGroup, integrationTestIPGroupName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create IP group: %v", err)
		}

		err = waitForIPGroupAvailable(ctx, ipGroupsClient, integrationTestResourceGroup, integrationTestIPGroupName)
		if err != nil {
			t.Fatalf("Failed waiting for IP group to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetIPGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving IP group %s in subscription %s, resource group %s",
				integrationTestIPGroupName, subscriptionID, integrationTestResourceGroup)

			ipGroupWrapper := manual.NewNetworkIPGroup(
				clients.NewIPGroupsClient(ipGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipGroupWrapper.Scopes()[0]

			ipGroupAdapter := sources.WrapperToAdapter(ipGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ipGroupAdapter.Get(ctx, scope, integrationTestIPGroupName, true)
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

			if uniqueAttrValue != integrationTestIPGroupName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestIPGroupName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved IP group %s", integrationTestIPGroupName)
		})

		t.Run("ListIPGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing IP groups in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			ipGroupWrapper := manual.NewNetworkIPGroup(
				clients.NewIPGroupsClient(ipGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipGroupWrapper.Scopes()[0]

			ipGroupAdapter := sources.WrapperToAdapter(ipGroupWrapper, sdpcache.NewNoOpCache())

			listable, ok := ipGroupAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list IP groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one IP group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestIPGroupName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find IP group %s in the list", integrationTestIPGroupName)
			}

			log.Printf("Found %d IP groups in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for IP group %s", integrationTestIPGroupName)

			ipGroupWrapper := manual.NewNetworkIPGroup(
				clients.NewIPGroupsClient(ipGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipGroupWrapper.Scopes()[0]

			ipGroupAdapter := sources.WrapperToAdapter(ipGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ipGroupAdapter.Get(ctx, scope, integrationTestIPGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkIPGroup.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkIPGroup, sdpItem.GetType())
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

			log.Printf("Verified item attributes for IP group %s", integrationTestIPGroupName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for IP group %s", integrationTestIPGroupName)

			ipGroupWrapper := manual.NewNetworkIPGroup(
				clients.NewIPGroupsClient(ipGroupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipGroupWrapper.Scopes()[0]

			ipGroupAdapter := sources.WrapperToAdapter(ipGroupWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := ipGroupAdapter.Get(ctx, scope, integrationTestIPGroupName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for IP group %s", len(linkedQueries), integrationTestIPGroupName)

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

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteIPGroup(ctx, ipGroupsClient, integrationTestResourceGroup, integrationTestIPGroupName)
		if err != nil {
			t.Fatalf("Failed to delete IP group: %v", err)
		}
	})
}

func createIPGroup(ctx context.Context, client *armnetwork.IPGroupsClient, resourceGroupName, ipGroupName, location string) error {
	existingIPGroup, err := client.Get(ctx, resourceGroupName, ipGroupName, nil)
	if err == nil {
		if existingIPGroup.Properties != nil && existingIPGroup.Properties.ProvisioningState != nil {
			state := *existingIPGroup.Properties.ProvisioningState
			if state == armnetwork.ProvisioningStateSucceeded {
				log.Printf("IP group %s already exists with state %s, skipping creation", ipGroupName, state)
				return nil
			}
			log.Printf("IP group %s exists but in state %s, will wait for it", ipGroupName, state)
		} else {
			log.Printf("IP group %s already exists, skipping creation", ipGroupName)
			return nil
		}
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, ipGroupName, armnetwork.IPGroup{
		Location: new(location),
		Properties: &armnetwork.IPGroupPropertiesFormat{
			IPAddresses: []*string{
				new("10.0.0.0/24"),
				new("192.168.1.1"),
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-ip-group"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("IP group %s already exists (conflict), skipping creation", ipGroupName)
			return nil
		}
		return fmt.Errorf("failed to begin creating IP group: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create IP group: %w", err)
	}

	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("IP group created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != armnetwork.ProvisioningStateSucceeded {
		return fmt.Errorf("IP group provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("IP group %s created successfully with provisioning state: %s", ipGroupName, provisioningState)
	return nil
}

func waitForIPGroupAvailable(ctx context.Context, client *armnetwork.IPGroupsClient, resourceGroupName, ipGroupName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for IP group %s to be available via API...", ipGroupName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, ipGroupName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("IP group %s not yet available (attempt %d/%d), waiting %v...", ipGroupName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking IP group availability: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armnetwork.ProvisioningStateSucceeded {
				log.Printf("IP group %s is available with provisioning state: %s", ipGroupName, state)
				return nil
			}
			if state == armnetwork.ProvisioningStateFailed {
				return fmt.Errorf("IP group provisioning failed with state: %s", state)
			}
			log.Printf("IP group %s provisioning state: %s (attempt %d/%d), waiting...", ipGroupName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		log.Printf("IP group %s is available", ipGroupName)
		return nil
	}

	return fmt.Errorf("timeout waiting for IP group %s to be available after %d attempts", ipGroupName, maxAttempts)
}

func deleteIPGroup(ctx context.Context, client *armnetwork.IPGroupsClient, resourceGroupName, ipGroupName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, ipGroupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("IP group %s not found, skipping deletion", ipGroupName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting IP group: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete IP group: %w", err)
	}

	log.Printf("IP group %s deleted successfully", ipGroupName)
	return nil
}
