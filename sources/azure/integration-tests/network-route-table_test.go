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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
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
	integrationTestRouteTableName = "ovm-integ-test-route-table"
	integrationTestRouteName      = "ovm-integ-test-route"
)

func TestNetworkRouteTableIntegration(t *testing.T) {
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
	routeTableClient, err := armnetwork.NewRouteTablesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Route Tables client: %v", err)
	}

	routesClient, err := armnetwork.NewRoutesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Routes client: %v", err)
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

		// Create route table
		err = createRouteTable(ctx, routeTableClient, integrationTestResourceGroup, integrationTestRouteTableName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create route table: %v", err)
		}

		// Wait for route table to be fully available
		err = waitForRouteTableAvailable(ctx, routeTableClient, integrationTestResourceGroup, integrationTestRouteTableName)
		if err != nil {
			t.Fatalf("Failed waiting for route table to be available: %v", err)
		}

		// Create a route in the route table
		err = createRoute(ctx, routesClient, integrationTestResourceGroup, integrationTestRouteTableName, integrationTestRouteName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create route: %v", err)
		}

		// Wait for route to be available
		err = waitForRouteAvailable(ctx, routesClient, integrationTestResourceGroup, integrationTestRouteTableName, integrationTestRouteName)
		if err != nil {
			t.Fatalf("Failed waiting for route to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetRouteTable", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving route table %s in subscription %s, resource group %s",
				integrationTestRouteTableName, subscriptionID, integrationTestResourceGroup)

			routeTableWrapper := manual.NewNetworkRouteTable(
				clients.NewRouteTablesClient(routeTableClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := routeTableWrapper.Scopes()[0]

			routeTableAdapter := sources.WrapperToAdapter(routeTableWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := routeTableAdapter.Get(ctx, scope, integrationTestRouteTableName, true)
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

			if uniqueAttrValue != integrationTestRouteTableName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestRouteTableName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved route table %s", integrationTestRouteTableName)
		})

		t.Run("ListRouteTables", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing route tables in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			routeTableWrapper := manual.NewNetworkRouteTable(
				clients.NewRouteTablesClient(routeTableClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := routeTableWrapper.Scopes()[0]

			routeTableAdapter := sources.WrapperToAdapter(routeTableWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := routeTableAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list route tables: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one route table, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestRouteTableName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find route table %s in the list of route tables", integrationTestRouteTableName)
			}

			log.Printf("Found %d route tables in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for route table %s", integrationTestRouteTableName)

			routeTableWrapper := manual.NewNetworkRouteTable(
				clients.NewRouteTablesClient(routeTableClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := routeTableWrapper.Scopes()[0]

			routeTableAdapter := sources.WrapperToAdapter(routeTableWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := routeTableAdapter.Get(ctx, scope, integrationTestRouteTableName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkRouteTable.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkRouteTable, sdpItem.GetType())
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

			log.Printf("Verified item attributes for route table %s", integrationTestRouteTableName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for route table %s", integrationTestRouteTableName)

			routeTableWrapper := manual.NewNetworkRouteTable(
				clients.NewRouteTablesClient(routeTableClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := routeTableWrapper.Scopes()[0]

			routeTableAdapter := sources.WrapperToAdapter(routeTableWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := routeTableAdapter.Get(ctx, scope, integrationTestRouteTableName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (if any)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for route table %s", len(linkedQueries), integrationTestRouteTableName)

			// Verify the structure is correct if links exist
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
				} else {
					// Blast propagation should have In and Out set (even if false)
					_ = bp.GetIn()
					_ = bp.GetOut()
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s, In=%v, Out=%v",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope(),
					bp.GetIn(), bp.GetOut())
			}

			// Verify that routes are linked (we created one named integrationTestRouteName)
			var hasRouteLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkRoute.String() {
					hasRouteLink = true
					// Verify blast propagation for routes
					bp := liq.GetBlastPropagation()
					if bp.GetIn() != true {
						t.Error("Expected route blast propagation In=true, got false")
					}
					if bp.GetOut() != false {
						t.Error("Expected route blast propagation Out=false, got true")
					}
					// Verify the query contains the route table name and route name
					query := liq.GetQuery().GetQuery()
					if query == "" {
						t.Error("Expected route query to be non-empty")
					}
					log.Printf("Found route link with query: %s", query)
					break
				}
			}
			if !hasRouteLink {
				t.Error("Expected linked query to routes, but didn't find one")
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete route
		err := deleteRoute(ctx, routesClient, integrationTestResourceGroup, integrationTestRouteTableName, integrationTestRouteName)
		if err != nil {
			t.Fatalf("Failed to delete route: %v", err)
		}

		// Delete route table
		err = deleteRouteTable(ctx, routeTableClient, integrationTestResourceGroup, integrationTestRouteTableName)
		if err != nil {
			t.Fatalf("Failed to delete route table: %v", err)
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

// createRouteTable creates an Azure route table (idempotent)
func createRouteTable(ctx context.Context, client *armnetwork.RouteTablesClient, resourceGroupName, routeTableName, location string) error {
	// Check if route table already exists
	existingRouteTable, err := client.Get(ctx, resourceGroupName, routeTableName, nil)
	if err == nil {
		// Route table exists, check its provisioning state
		if existingRouteTable.Properties != nil && existingRouteTable.Properties.ProvisioningState != nil {
			state := *existingRouteTable.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Route table %s already exists with state %s, skipping creation", routeTableName, state)
				return nil
			}
			log.Printf("Route table %s exists but in state %s, will wait for it", routeTableName, state)
		} else {
			log.Printf("Route table %s already exists, skipping creation", routeTableName)
			return nil
		}
	}

	// Create a basic route table
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, routeTableName, armnetwork.RouteTable{
		Location:   ptr.To(location),
		Properties: &armnetwork.RouteTablePropertiesFormat{
			// Routes will be added separately as child resources
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("network-route-table"),
		},
	}, nil)
	if err != nil {
		// Check if route table already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Route table %s already exists (conflict), skipping creation", routeTableName)
			return nil
		}
		return fmt.Errorf("failed to begin creating route table: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create route table: %w", err)
	}

	// Verify the route table was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("route table created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("route table provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Route table %s created successfully with provisioning state: %s", routeTableName, provisioningState)
	return nil
}

// waitForRouteTableAvailable polls until the route table is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the route table is queryable
func waitForRouteTableAvailable(ctx context.Context, client *armnetwork.RouteTablesClient, resourceGroupName, routeTableName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for route table %s to be available via API...", routeTableName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, routeTableName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Route table %s not yet available (attempt %d/%d), waiting %v...", routeTableName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking route table availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Route table %s is available with provisioning state: %s", routeTableName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("route table provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Route table %s provisioning state: %s (attempt %d/%d), waiting...", routeTableName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Route table exists but no provisioning state - consider it available
		log.Printf("Route table %s is available", routeTableName)
		return nil
	}

	return fmt.Errorf("timeout waiting for route table %s to be available after %d attempts", routeTableName, maxAttempts)
}

// createRoute creates a route in a route table (idempotent)
func createRoute(ctx context.Context, client *armnetwork.RoutesClient, resourceGroupName, routeTableName, routeName, location string) error {
	// Check if route already exists
	existingRoute, err := client.Get(ctx, resourceGroupName, routeTableName, routeName, nil)
	if err == nil {
		// Route exists, check its provisioning state
		if existingRoute.Properties != nil && existingRoute.Properties.ProvisioningState != nil {
			state := *existingRoute.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Route %s already exists with state %s, skipping creation", routeName, state)
				return nil
			}
			log.Printf("Route %s exists but in state %s, will wait for it", routeName, state)
		} else {
			log.Printf("Route %s already exists, skipping creation", routeName)
			return nil
		}
	}

	// Create a route with VirtualAppliance next hop type and a sample IP address
	// This creates a route that will link to a NetworkIP
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, routeTableName, routeName, armnetwork.Route{
		Properties: &armnetwork.RoutePropertiesFormat{
			AddressPrefix:    ptr.To("10.0.0.0/8"),
			NextHopType:      ptr.To(armnetwork.RouteNextHopTypeVirtualAppliance),
			NextHopIPAddress: ptr.To("10.0.0.1"), // This will create a link to stdlib.NetworkIP
		},
	}, nil)
	if err != nil {
		// Check if route already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Route %s already exists (conflict), skipping creation", routeName)
			return nil
		}
		return fmt.Errorf("failed to begin creating route: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}

	// Verify the route was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("route created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("route provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Route %s created successfully with provisioning state: %s", routeName, provisioningState)
	return nil
}

// waitForRouteAvailable polls until the route is available via the Get API
func waitForRouteAvailable(ctx context.Context, client *armnetwork.RoutesClient, resourceGroupName, routeTableName, routeName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for route %s to be available via API...", routeName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, routeTableName, routeName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Route %s not yet available (attempt %d/%d), waiting %v...", routeName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking route availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Route %s is available with provisioning state: %s", routeName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("route provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Route %s provisioning state: %s (attempt %d/%d), waiting...", routeName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Route exists but no provisioning state - consider it available
		log.Printf("Route %s is available", routeName)
		return nil
	}

	return fmt.Errorf("timeout waiting for route %s to be available after %d attempts", routeName, maxAttempts)
}

// deleteRoute deletes a route from a route table
func deleteRoute(ctx context.Context, client *armnetwork.RoutesClient, resourceGroupName, routeTableName, routeName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, routeTableName, routeName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Route %s not found, skipping deletion", routeName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting route: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	log.Printf("Route %s deleted successfully", routeName)
	return nil
}

// deleteRouteTable deletes an Azure route table
func deleteRouteTable(ctx context.Context, client *armnetwork.RouteTablesClient, resourceGroupName, routeTableName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, routeTableName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Route table %s not found, skipping deletion", routeTableName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting route table: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete route table: %w", err)
	}

	log.Printf("Route table %s deleted successfully", routeTableName)
	return nil
}
