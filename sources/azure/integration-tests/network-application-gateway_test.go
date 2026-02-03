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
	integrationTestAGName            = "ovm-integ-test-ag"
	integrationTestVNetNameForAG     = "ovm-integ-test-vnet-for-ag"
	integrationTestAGSubnetName      = "ag-subnet"
	integrationTestPublicIPNameForAG = "ovm-integ-test-public-ip-for-ag"
)

func TestNetworkApplicationGatewayIntegration(t *testing.T) {
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
	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Subnets client: %v", err)
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	agClient, err := armnetwork.NewApplicationGatewaysClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Application Gateways client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network for the application gateway
		// Application Gateway requires a dedicated subnet
		err = createVirtualNetworkForAG(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForAG, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Create dedicated subnet for Application Gateway
		err = createAGSubnet(ctx, subnetClient, integrationTestResourceGroup, integrationTestVNetNameForAG, integrationTestAGSubnetName)
		if err != nil {
			t.Fatalf("Failed to create Application Gateway subnet: %v", err)
		}

		// Create public IP address for the application gateway (needed even if AG exists)
		err = createPublicIPForAG(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForAG, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		// Check if Application Gateway already exists first (quick check)
		existingAG, err := agClient.Get(ctx, integrationTestResourceGroup, integrationTestAGName, nil)
		if err == nil {
			// Application Gateway exists, check if it's ready
			if existingAG.Properties != nil && existingAG.Properties.ProvisioningState != nil {
				state := *existingAG.Properties.ProvisioningState
				if state == "Succeeded" {
					log.Printf("Application Gateway %s already exists and is ready, skipping creation", integrationTestAGName)
				} else {
					log.Printf("Application Gateway %s exists but in state %s, waiting for it to be ready", integrationTestAGName, state)
					err = waitForApplicationGatewayAvailable(ctx, agClient, integrationTestResourceGroup, integrationTestAGName)
					if err != nil {
						t.Fatalf("Failed waiting for existing application gateway to be ready: %v", err)
					}
				}
			} else {
				log.Printf("Application Gateway %s already exists, verifying availability", integrationTestAGName)
				err = waitForApplicationGatewayAvailable(ctx, agClient, integrationTestResourceGroup, integrationTestAGName)
				if err != nil {
					t.Fatalf("Failed waiting for application gateway to be available: %v", err)
				}
			}
		} else {
			// Application Gateway doesn't exist
			// Application Gateway creation takes 15-20 minutes which exceeds test timeout
			// For integration tests, we require the Application Gateway to already exist
			log.Printf("Application Gateway %s does not exist", integrationTestAGName)
			log.Printf("Application Gateway creation takes 15-20 minutes, which exceeds the test timeout of 5 minutes.")
			log.Printf("Please create the Application Gateway manually or wait for a previous creation to complete.")
			log.Printf("Required resources should be ready: subnet %s and public IP %s", integrationTestAGSubnetName, integrationTestPublicIPNameForAG)
			t.Skipf("Application Gateway %s does not exist. Please create it first (takes 15-20 minutes) or ensure it exists in 'Succeeded' state before running integration tests", integrationTestAGName)
		}

		log.Printf("Setup completed: Application Gateway %s is available", integrationTestAGName)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetApplicationGateway", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving application gateway %s in subscription %s, resource group %s",
				integrationTestAGName, subscriptionID, integrationTestResourceGroup)

			agWrapper := manual.NewNetworkApplicationGateway(
				clients.NewApplicationGatewaysClient(agClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := agWrapper.Scopes()[0]

			agAdapter := sources.WrapperToAdapter(agWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := agAdapter.Get(ctx, scope, integrationTestAGName, true)
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

			if uniqueAttrValue != integrationTestAGName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestAGName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.NetworkApplicationGateway.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.NetworkApplicationGateway, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved application gateway %s", integrationTestAGName)
		})

		t.Run("ListApplicationGateways", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing application gateways in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			agWrapper := manual.NewNetworkApplicationGateway(
				clients.NewApplicationGatewaysClient(agClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := agWrapper.Scopes()[0]

			agAdapter := sources.WrapperToAdapter(agWrapper, sdpcache.NewNoOpCache())
			listable, ok := agAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least 1 application gateway, got: %d", len(sdpItems))
			}

			// Find our test application gateway
			found := false
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestAGName {
					found = true
					if item.GetType() != azureshared.NetworkApplicationGateway.String() {
						t.Errorf("Expected type %s, got %s", azureshared.NetworkApplicationGateway, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find application gateway %s in list, but didn't", integrationTestAGName)
			}

			log.Printf("Successfully listed %d application gateways", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			agWrapper := manual.NewNetworkApplicationGateway(
				clients.NewApplicationGatewaysClient(agClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := agWrapper.Scopes()[0]

			agAdapter := sources.WrapperToAdapter(agWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := agAdapter.Get(ctx, scope, integrationTestAGName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkApplicationGateway.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkApplicationGateway, sdpItem.GetType())
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

			log.Printf("Verified item attributes for application gateway %s", integrationTestAGName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			agWrapper := manual.NewNetworkApplicationGateway(
				clients.NewApplicationGatewaysClient(agClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := agWrapper.Scopes()[0]

			agAdapter := sources.WrapperToAdapter(agWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := agAdapter.Get(ctx, scope, integrationTestAGName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected linked item types for application gateway
			expectedLinkedTypes := map[string]bool{
				azureshared.NetworkApplicationGatewayGatewayIPConfiguration.String():  false,
				azureshared.NetworkApplicationGatewayFrontendIPConfiguration.String(): false,
				azureshared.NetworkApplicationGatewayBackendAddressPool.String():      false,
				azureshared.NetworkApplicationGatewayHTTPListener.String():            false,
				azureshared.NetworkApplicationGatewayBackendHTTPSettings.String():     false,
				azureshared.NetworkApplicationGatewayRequestRoutingRule.String():      false,
				azureshared.NetworkPublicIPAddress.String():                           false,
				azureshared.NetworkSubnet.String():                                    false,
				azureshared.NetworkVirtualNetwork.String():                            false,
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				linkedType := query.GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true
				}

				// Verify query has required fields
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

				// Verify blast propagation is set
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
				} else {
					// Blast propagation should have In and Out set (even if false)
					_ = bp.GetIn()
					_ = bp.GetOut()
				}
			}

			// Verify critical linked types were found
			criticalTypes := []string{
				azureshared.NetworkApplicationGatewayGatewayIPConfiguration.String(),
				azureshared.NetworkApplicationGatewayFrontendIPConfiguration.String(),
				azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
				azureshared.NetworkApplicationGatewayHTTPListener.String(),
				azureshared.NetworkApplicationGatewayBackendHTTPSettings.String(),
				azureshared.NetworkApplicationGatewayRequestRoutingRule.String(),
			}

			for _, linkedType := range criticalTypes {
				if !expectedLinkedTypes[linkedType] {
					t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
				}
			}

			log.Printf("Verified %d linked item queries for application gateway %s", len(linkedQueries), integrationTestAGName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete application gateway
		err := deleteApplicationGateway(ctx, agClient, integrationTestResourceGroup, integrationTestAGName)
		if err != nil {
			t.Fatalf("Failed to delete application gateway: %v", err)
		}

		// Delete public IP address
		err = deletePublicIPForAG(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForAG)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForAG(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForAG)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

// createVirtualNetworkForAG creates an Azure virtual network for Application Gateway (idempotent)
func createVirtualNetworkForAG(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	// Check if VNet already exists
	_, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", vnetName)
		return nil
	}

	// Create the VNet
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: ptr.To(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{ptr.To("10.3.0.0/16")},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating virtual network: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual network: %w", err)
	}

	log.Printf("Virtual network %s created successfully", vnetName)
	return nil
}

// createAGSubnet creates a dedicated subnet for Application Gateway (idempotent)
// Application Gateway requires a dedicated subnet with at least /24 address space
func createAGSubnet(ctx context.Context, client *armnetwork.SubnetsClient, resourceGroupName, vnetName, subnetName string) error {
	// Check if subnet already exists
	_, err := client.Get(ctx, resourceGroupName, vnetName, subnetName, nil)
	if err == nil {
		log.Printf("Subnet %s already exists, skipping creation", subnetName)
		return nil
	}

	// Create the subnet with /24 address space for Application Gateway
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, subnetName, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: ptr.To("10.3.0.0/24"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Subnet %s already exists (conflict), skipping creation", subnetName)
			return nil
		}
		return fmt.Errorf("failed to begin creating subnet: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	log.Printf("Subnet %s created successfully", subnetName)
	return nil
}

// deleteVirtualNetworkForAG deletes an Azure virtual network
func deleteVirtualNetworkForAG(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, vnetName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual network %s not found, skipping deletion", vnetName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting virtual network: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network: %w", err)
	}

	log.Printf("Virtual network %s deleted successfully", vnetName)
	return nil
}

// createPublicIPForAG creates an Azure public IP address for Application Gateway (idempotent)
func createPublicIPForAG(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName, location string) error {
	// Check if public IP already exists
	_, err := client.Get(ctx, resourceGroupName, publicIPName, nil)
	if err == nil {
		log.Printf("Public IP address %s already exists, skipping creation", publicIPName)
		return nil
	}

	// Create the public IP address with Standard SKU (required for Application Gateway v2)
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPName, armnetwork.PublicIPAddress{
		Location: ptr.To(location),
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: ptr.To(armnetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:   ptr.To(armnetwork.IPVersionIPv4),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: ptr.To(armnetwork.PublicIPAddressSKUNameStandard),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating public IP address: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create public IP address: %w", err)
	}

	log.Printf("Public IP address %s created successfully", publicIPName)
	return nil
}

// deletePublicIPForAG deletes an Azure public IP address
func deletePublicIPForAG(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, publicIPName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Public IP address %s not found, skipping deletion", publicIPName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting public IP address: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete public IP address: %w", err)
	}

	log.Printf("Public IP address %s deleted successfully", publicIPName)
	return nil
}

// waitForApplicationGatewayAvailable polls until the application gateway is available via the Get API
func waitForApplicationGatewayAvailable(ctx context.Context, client *armnetwork.ApplicationGatewaysClient, resourceGroupName, agName string) error {
	maxAttempts := 30 // Application Gateways take longer to provision
	pollInterval := 10 * time.Second

	log.Printf("Waiting for application gateway %s to be available via API...", agName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, agName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Application Gateway %s not yet available (attempt %d/%d), waiting %v...", agName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking application gateway availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Application Gateway %s is available with provisioning state: %s", agName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("application gateway provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Application Gateway %s provisioning state: %s (attempt %d/%d), waiting...", agName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Application Gateway exists but no provisioning state - consider it available
		log.Printf("Application Gateway %s is available", agName)
		return nil
	}

	return fmt.Errorf("timeout waiting for application gateway %s to be available after %d attempts", agName, maxAttempts)
}

// deleteApplicationGateway deletes an Azure Application Gateway
func deleteApplicationGateway(ctx context.Context, client *armnetwork.ApplicationGatewaysClient, resourceGroupName, agName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, agName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Application Gateway %s not found, skipping deletion", agName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting application gateway: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete application gateway: %w", err)
	}

	log.Printf("Application Gateway %s deleted successfully", agName)
	return nil
}
