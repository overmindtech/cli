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
	integrationTestLocalNetworkGatewayName = "ovm-integ-test-lng"
)

func TestNetworkLocalNetworkGatewayIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	localNetworkGatewaysClient, err := armnetwork.NewLocalNetworkGatewaysClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Local Network Gateways client: %v", err)
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

		err = createLocalNetworkGateway(ctx, localNetworkGatewaysClient, integrationTestResourceGroup, integrationTestLocalNetworkGatewayName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create local network gateway: %v", err)
		}

		err = waitForLocalNetworkGatewayAvailable(ctx, localNetworkGatewaysClient, integrationTestResourceGroup, integrationTestLocalNetworkGatewayName)
		if err != nil {
			t.Fatalf("Failed waiting for local network gateway to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetLocalNetworkGateway", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving local network gateway %s in subscription %s, resource group %s",
				integrationTestLocalNetworkGatewayName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkLocalNetworkGateway(
				clients.NewLocalNetworkGatewaysClient(localNetworkGatewaysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestLocalNetworkGatewayName, true)
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

			if uniqueAttrValue != integrationTestLocalNetworkGatewayName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestLocalNetworkGatewayName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved local network gateway %s", integrationTestLocalNetworkGatewayName)
		})

		t.Run("ListLocalNetworkGateways", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing local network gateways in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkLocalNetworkGateway(
				clients.NewLocalNetworkGatewaysClient(localNetworkGatewaysClient),
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
				t.Fatalf("Failed to list local network gateways: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one local network gateway, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestLocalNetworkGatewayName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find local network gateway %s in the list", integrationTestLocalNetworkGatewayName)
			}

			log.Printf("Found %d local network gateways in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for local network gateway %s", integrationTestLocalNetworkGatewayName)

			wrapper := manual.NewNetworkLocalNetworkGateway(
				clients.NewLocalNetworkGatewaysClient(localNetworkGatewaysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestLocalNetworkGatewayName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLocalNetworkGateway.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkLocalNetworkGateway, sdpItem.GetType())
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

			log.Printf("Verified item attributes for local network gateway %s", integrationTestLocalNetworkGatewayName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for local network gateway %s", integrationTestLocalNetworkGatewayName)

			wrapper := manual.NewNetworkLocalNetworkGateway(
				clients.NewLocalNetworkGatewaysClient(localNetworkGatewaysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestLocalNetworkGatewayName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for local network gateway %s", len(linkedQueries), integrationTestLocalNetworkGatewayName)

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

		err := deleteLocalNetworkGateway(ctx, localNetworkGatewaysClient, integrationTestResourceGroup, integrationTestLocalNetworkGatewayName)
		if err != nil {
			t.Fatalf("Failed to delete local network gateway: %v", err)
		}
	})
}

func createLocalNetworkGateway(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName, location string) error {
	existingGateway, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
	if err == nil {
		if existingGateway.Properties != nil && existingGateway.Properties.ProvisioningState != nil {
			state := string(*existingGateway.Properties.ProvisioningState)
			if state == "Succeeded" {
				log.Printf("Local network gateway %s already exists with state %s, skipping creation", gatewayName, state)
				return nil
			}
			log.Printf("Local network gateway %s exists but in state %s, will wait for it", gatewayName, state)
		} else {
			log.Printf("Local network gateway %s already exists, skipping creation", gatewayName)
			return nil
		}
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, gatewayName, armnetwork.LocalNetworkGateway{
		Location: new(location),
		Properties: &armnetwork.LocalNetworkGatewayPropertiesFormat{
			GatewayIPAddress: new("203.0.113.1"),
			LocalNetworkAddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					new("10.1.0.0/16"),
					new("10.2.0.0/16"),
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-local-network-gateway"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, gatewayName, nil); getErr == nil {
				log.Printf("Local network gateway %s already exists (conflict), skipping creation", gatewayName)
				return nil
			}
			return fmt.Errorf("local network gateway %s conflict but not retrievable: %w", gatewayName, err)
		}
		return fmt.Errorf("failed to begin creating local network gateway: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create local network gateway: %w", err)
	}

	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("local network gateway created but provisioning state is unknown")
	}

	provisioningState := string(*resp.Properties.ProvisioningState)
	if provisioningState != "Succeeded" {
		return fmt.Errorf("local network gateway provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Local network gateway %s created successfully with provisioning state: %s", gatewayName, provisioningState)
	return nil
}

func waitForLocalNetworkGatewayAvailable(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for local network gateway %s to be available via API...", gatewayName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Local network gateway %s not yet available (attempt %d/%d), waiting %v...", gatewayName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking local network gateway availability: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := string(*resp.Properties.ProvisioningState)
			if state == "Succeeded" {
				log.Printf("Local network gateway %s is available with provisioning state: %s", gatewayName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("local network gateway provisioning failed with state: %s", state)
			}
			log.Printf("Local network gateway %s provisioning state: %s (attempt %d/%d), waiting...", gatewayName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		log.Printf("Local network gateway %s is available", gatewayName)
		return nil
	}

	return fmt.Errorf("timeout waiting for local network gateway %s to be available after %d attempts", gatewayName, maxAttempts)
}

func deleteLocalNetworkGateway(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, gatewayName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Local network gateway %s not found, skipping deletion", gatewayName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting local network gateway: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete local network gateway: %w", err)
	}

	log.Printf("Local network gateway %s deleted successfully", gatewayName)
	return nil
}
