package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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
	// Azure only allows one Network Watcher per region per subscription.
	// We create a test Network Watcher in our integration test resource group.
	integrationTestNetworkWatcherTestName = "ovm-integ-test-nw"
)

func TestNetworkNetworkWatcherIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	networkWatchersClient, err := armnetwork.NewWatchersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Network Watchers client: %v", err)
	}

	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create network watcher - Azure only allows one per region per subscription
		err = createNetworkWatcher(ctx, networkWatchersClient, integrationTestResourceGroup, integrationTestNetworkWatcherTestName, integrationTestLocation)
		if err != nil {
			// If we hit the limit, it means a Network Watcher already exists in another RG
			if strings.Contains(err.Error(), "NetworkWatcherCountLimitReached") {
				t.Skipf("Skipping: Azure allows only one Network Watcher per region. One already exists: %v", err)
			}
			t.Fatalf("Failed to create network watcher: %v", err)
		}

		// Wait for network watcher to be available
		err = waitForNetworkWatcherAvailable(ctx, networkWatchersClient, integrationTestResourceGroup, integrationTestNetworkWatcherTestName)
		if err != nil {
			t.Fatalf("Failed waiting for network watcher: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetNetworkWatcher", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving network watcher %s in subscription %s, resource group %s",
				integrationTestNetworkWatcherTestName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkNetworkWatcher(
				clients.NewNetworkWatchersClient(networkWatchersClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestNetworkWatcherTestName, true)
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

			if uniqueAttrValue != integrationTestNetworkWatcherTestName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestNetworkWatcherTestName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved network watcher %s", integrationTestNetworkWatcherTestName)
		})

		t.Run("ListNetworkWatchers", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing network watchers in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkNetworkWatcher(
				clients.NewNetworkWatchersClient(networkWatchersClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			listable, ok := adapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list network watchers: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one network watcher, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestNetworkWatcherTestName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find network watcher %s in the list", integrationTestNetworkWatcherTestName)
			}

			log.Printf("Found %d network watchers in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for network watcher %s", integrationTestNetworkWatcherTestName)

			wrapper := manual.NewNetworkNetworkWatcher(
				clients.NewNetworkWatchersClient(networkWatchersClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestNetworkWatcherTestName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()

			for _, query := range linkedQueries {
				q := query.GetQuery()
				if q == nil {
					t.Error("LinkedItemQuery has nil Query")
					continue
				}

				if q.GetType() == "" {
					t.Error("LinkedItemQuery has empty Type")
				}

				if q.GetMethod() != sdp.QueryMethod_GET && q.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("LinkedItemQuery has invalid Method: %v", q.GetMethod())
				}

				if q.GetQuery() == "" {
					t.Error("LinkedItemQuery has empty Query")
				}

				if q.GetScope() == "" {
					t.Error("LinkedItemQuery has empty Scope")
				}
			}

			log.Printf("Verified %d linked item queries for network watcher %s", len(linkedQueries), integrationTestNetworkWatcherTestName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for network watcher %s", integrationTestNetworkWatcherTestName)

			wrapper := manual.NewNetworkNetworkWatcher(
				clients.NewNetworkWatchersClient(networkWatchersClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestNetworkWatcherTestName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkNetworkWatcher.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkNetworkWatcher, sdpItem.GetType())
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

			log.Printf("Verified item attributes for network watcher %s", integrationTestNetworkWatcherTestName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete the network watcher we created
		err := deleteNetworkWatcher(ctx, networkWatchersClient, integrationTestResourceGroup, integrationTestNetworkWatcherTestName)
		if err != nil {
			t.Logf("Warning: Failed to delete network watcher %s: %v", integrationTestNetworkWatcherTestName, err)
		}
	})
}

func createNetworkWatcher(ctx context.Context, client *armnetwork.WatchersClient, resourceGroup, name, location string) error {
	_, err := client.Get(ctx, resourceGroup, name, nil)
	if err == nil {
		log.Printf("Network watcher %s already exists, skipping creation", name)
		return nil
	}

	result, err := client.CreateOrUpdate(ctx, resourceGroup, name, armnetwork.Watcher{
		Location: &location,
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroup, name, nil); getErr == nil {
				log.Printf("Network watcher %s already exists (conflict), skipping", name)
				return nil
			}
			return fmt.Errorf("network watcher %s conflict but not retrievable: %w", name, err)
		}
		return fmt.Errorf("failed to create network watcher: %w", err)
	}

	log.Printf("Network watcher %s created: %v", name, result.Watcher.Name)
	return nil
}

func waitForNetworkWatcherAvailable(ctx context.Context, client *armnetwork.WatchersClient, resourceGroup, name string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroup, name, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("network watcher %s not found after %d attempts", name, notFoundCount)
				}
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking network watcher: %w", err)
		}
		notFoundCount = 0
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armnetwork.ProvisioningStateSucceeded {
			log.Printf("Network watcher %s is available", name)
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for network watcher %s", name)
}

func deleteNetworkWatcher(ctx context.Context, client *armnetwork.WatchersClient, resourceGroup, name string) error {
	poller, err := client.BeginDelete(ctx, resourceGroup, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Network watcher %s already deleted", name)
			return nil
		}
		return fmt.Errorf("failed to begin delete network watcher: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete network watcher: %w", err)
	}

	log.Printf("Network watcher %s deleted successfully", name)
	return nil
}
