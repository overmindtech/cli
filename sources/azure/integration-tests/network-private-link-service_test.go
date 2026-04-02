package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestPLSName           = "ovm-integ-test-pls"
	integrationTestVNetNameForPLS    = "ovm-integ-test-vnet-for-pls"
	integrationTestSubnetNameForPLS  = "pls-subnet"
	integrationTestLBNameForPLS      = "ovm-integ-test-lb-for-pls"
	integrationTestFrontendIPForPLS  = "frontend-ip-config"
	integrationTestBackendPoolForPLS = "backend-pool"
)

func TestNetworkPrivateLinkServiceIntegration(t *testing.T) {
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

	lbClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancers client: %v", err)
	}

	plsClient, err := armnetwork.NewPrivateLinkServicesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Private Link Services client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network for private link service (with special subnet settings)
		err = createVirtualNetworkForPLS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForPLS, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for load balancer and private link service
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetNameForPLS, integrationTestSubnetNameForPLS, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create internal load balancer for private link service
		err = createInternalLoadBalancerForPLS(ctx, lbClient, subscriptionID, integrationTestResourceGroup, integrationTestLBNameForPLS, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create internal load balancer: %v", err)
		}

		// Get load balancer frontend IP configuration ID
		lbResp, err := lbClient.Get(ctx, integrationTestResourceGroup, integrationTestLBNameForPLS, nil)
		if err != nil {
			t.Fatalf("Failed to get load balancer: %v", err)
		}

		var frontendIPConfigID string
		if lbResp.Properties != nil && len(lbResp.Properties.FrontendIPConfigurations) > 0 {
			frontendIPConfigID = *lbResp.Properties.FrontendIPConfigurations[0].ID
		}
		if frontendIPConfigID == "" {
			t.Fatalf("Failed to get frontend IP configuration ID")
		}

		// Create private link service
		err = createPrivateLinkService(ctx, plsClient, integrationTestResourceGroup, integrationTestPLSName, integrationTestLocation, *subnetResp.ID, frontendIPConfigID)
		if err != nil {
			t.Fatalf("Failed to create private link service: %v", err)
		}

		setupCompleted = true
		log.Printf("Setup completed: Private Link Service %s created", integrationTestPLSName)
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetPrivateLinkService", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving private link service %s in subscription %s, resource group %s",
				integrationTestPLSName, subscriptionID, integrationTestResourceGroup)

			plsWrapper := manual.NewNetworkPrivateLinkService(
				clients.NewPrivateLinkServicesClient(plsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := plsWrapper.Scopes()[0]

			plsAdapter := sources.WrapperToAdapter(plsWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := plsAdapter.Get(ctx, scope, integrationTestPLSName, true)
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

			if uniqueAttrValue != integrationTestPLSName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestPLSName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.NetworkPrivateLinkService.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.NetworkPrivateLinkService, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved private link service %s", integrationTestPLSName)
		})

		t.Run("ListPrivateLinkServices", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing private link services in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			plsWrapper := manual.NewNetworkPrivateLinkService(
				clients.NewPrivateLinkServicesClient(plsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := plsWrapper.Scopes()[0]

			plsAdapter := sources.WrapperToAdapter(plsWrapper, sdpcache.NewNoOpCache())
			listable, ok := plsAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least 1 private link service, got: %d", len(sdpItems))
			}

			// Find our test private link service
			found := false
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == integrationTestPLSName {
						found = true
						if item.GetType() != azureshared.NetworkPrivateLinkService.String() {
							t.Errorf("Expected type %s, got %s", azureshared.NetworkPrivateLinkService, item.GetType())
						}
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find private link service %s in list, but didn't", integrationTestPLSName)
			}

			log.Printf("Successfully listed %d private link services", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			plsWrapper := manual.NewNetworkPrivateLinkService(
				clients.NewPrivateLinkServicesClient(plsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := plsWrapper.Scopes()[0]

			plsAdapter := sources.WrapperToAdapter(plsWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := plsAdapter.Get(ctx, scope, integrationTestPLSName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkPrivateLinkService.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkPrivateLinkService, sdpItem.GetType())
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

			// Verify Validate() passes
			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Expected item to validate, got error: %v", err)
			}

			log.Printf("Verified item attributes for private link service %s", integrationTestPLSName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			plsWrapper := manual.NewNetworkPrivateLinkService(
				clients.NewPrivateLinkServicesClient(plsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := plsWrapper.Scopes()[0]

			plsAdapter := sources.WrapperToAdapter(plsWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := plsAdapter.Get(ctx, scope, integrationTestPLSName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify each linked item query has required fields
			for i, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Errorf("Linked query %d has empty Type", i)
				}
				if query.GetQuery() == "" {
					t.Errorf("Linked query %d has empty Query", i)
				}
				if query.GetScope() == "" {
					t.Errorf("Linked query %d has empty Scope", i)
				}
			}

			// Verify expected linked item types
			expectedLinkedTypes := map[string]bool{
				azureshared.NetworkSubnet.String():                              false,
				azureshared.NetworkVirtualNetwork.String():                      false,
				azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(): false,
				azureshared.NetworkLoadBalancer.String():                        false,
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true
				}
			}

			for linkedType, found := range expectedLinkedTypes {
				if !found {
					t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
				}
			}

			log.Printf("Verified %d linked item queries for private link service %s", len(linkedQueries), integrationTestPLSName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete private link service
		err := deletePrivateLinkService(ctx, plsClient, integrationTestResourceGroup, integrationTestPLSName)
		if err != nil {
			t.Logf("Warning: Failed to delete private link service: %v", err)
		}

		// Delete load balancer
		err = deleteLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestLBNameForPLS)
		if err != nil {
			t.Logf("Warning: Failed to delete load balancer: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForPLS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForPLS)
		if err != nil {
			t.Logf("Warning: Failed to delete virtual network: %v", err)
		}

		log.Printf("Teardown completed")
	})
}

// createVirtualNetworkForPLS creates an Azure virtual network with a subnet that has privateLinkServiceNetworkPolicies disabled
func createVirtualNetworkForPLS(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	// Check if VNet already exists
	_, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", vnetName)
		return nil
	}

	// Create the VNet with a subnet that has privateLinkServiceNetworkPolicies disabled
	disabled := armnetwork.VirtualNetworkPrivateLinkServiceNetworkPoliciesDisabled
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.3.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: new(integrationTestSubnetNameForPLS),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix:                     new("10.3.0.0/24"),
						PrivateLinkServiceNetworkPolicies: &disabled,
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
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

// deleteVirtualNetworkForPLS deletes an Azure virtual network
func deleteVirtualNetworkForPLS(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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

// createInternalLoadBalancerForPLS creates an Azure internal load balancer for private link service
func createInternalLoadBalancerForPLS(ctx context.Context, client *armnetwork.LoadBalancersClient, subscriptionID, resourceGroupName, lbName, location, subnetID string) error {
	// Check if load balancer already exists
	_, err := client.Get(ctx, resourceGroupName, lbName, nil)
	if err == nil {
		log.Printf("Load balancer %s already exists, skipping creation", lbName)
		return nil
	}

	// Create the internal load balancer
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, lbName, armnetwork.LoadBalancer{
		Location: new(location),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: new(integrationTestFrontendIPForPLS),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: new(subnetID),
						},
						PrivateIPAllocationMethod: new(armnetwork.IPAllocationMethodDynamic),
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: new(integrationTestBackendPoolForPLS),
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: new("lb-rule"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s",
								subscriptionID, resourceGroupName, lbName, integrationTestFrontendIPForPLS)),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s",
								subscriptionID, resourceGroupName, lbName, integrationTestBackendPoolForPLS)),
						},
						Protocol:             new(armnetwork.TransportProtocolTCP),
						FrontendPort:         new(int32(80)),
						BackendPort:          new(int32(80)),
						EnableFloatingIP:     new(false),
						IdleTimeoutInMinutes: new(int32(4)),
					},
				},
			},
		},
		SKU: &armnetwork.LoadBalancerSKU{
			Name: new(armnetwork.LoadBalancerSKUNameStandard),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating load balancer: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	log.Printf("Load balancer %s created successfully", lbName)
	return nil
}

// createPrivateLinkService creates an Azure Private Link Service
func createPrivateLinkService(ctx context.Context, client *armnetwork.PrivateLinkServicesClient, resourceGroupName, plsName, location, subnetID, frontendIPConfigID string) error {
	// Check if private link service already exists
	_, err := client.Get(ctx, resourceGroupName, plsName, nil)
	if err == nil {
		log.Printf("Private link service %s already exists, skipping creation", plsName)
		return nil
	}

	// Create the private link service
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, plsName, armnetwork.PrivateLinkService{
		Location: new(location),
		Properties: &armnetwork.PrivateLinkServiceProperties{
			LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					ID: new(frontendIPConfigID),
				},
			},
			IPConfigurations: []*armnetwork.PrivateLinkServiceIPConfiguration{
				{
					Name: new("pls-ip-config"),
					Properties: &armnetwork.PrivateLinkServiceIPConfigurationProperties{
						Subnet: &armnetwork.Subnet{
							ID: new(subnetID),
						},
						PrivateIPAllocationMethod: new(armnetwork.IPAllocationMethodDynamic),
						Primary:                   new(true),
					},
				},
			},
			EnableProxyProtocol: new(false),
			Fqdns: []*string{
				new("test-pls.example.com"),
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			// Verify the resource actually exists before treating conflict as success
			if _, getErr := client.Get(ctx, resourceGroupName, plsName, nil); getErr == nil {
				log.Printf("Private link service %s already exists (conflict), skipping creation", plsName)
				return nil
			}
			return fmt.Errorf("private link service %s conflict but not retrievable: %w", plsName, err)
		}
		return fmt.Errorf("failed to begin creating private link service: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create private link service: %w", err)
	}

	log.Printf("Private link service %s created successfully", plsName)
	return nil
}

// deletePrivateLinkService deletes an Azure Private Link Service
func deletePrivateLinkService(ctx context.Context, client *armnetwork.PrivateLinkServicesClient, resourceGroupName, plsName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, plsName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Private link service %s not found, skipping deletion", plsName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting private link service: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete private link service: %w", err)
	}

	log.Printf("Private link service %s deleted successfully", plsName)
	return nil
}
