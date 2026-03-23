package integrationtests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
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
	integrationTestBackendPoolLBName          = "ovm-integ-test-lb-for-pool"
	integrationTestBackendPoolName            = "test-backend-pool"
	integrationTestVNetNameForBackendPool     = "ovm-integ-test-vnet-for-pool"
	integrationTestSubnetNameForBackendPool   = "default"
	integrationTestPublicIPNameForBackendPool = "ovm-integ-test-pip-for-pool"
)

func TestNetworkLoadBalancerBackendAddressPoolIntegration(t *testing.T) {
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

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	lbClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancers client: %v", err)
	}

	backendPoolClient, err := armnetwork.NewLoadBalancerBackendAddressPoolsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancer Backend Address Pools client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createVirtualNetworkForBackendPool(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForBackendPool, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		err = createPublicIPForBackendPool(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForBackendPool, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		publicIPResp, err := publicIPClient.Get(ctx, integrationTestResourceGroup, integrationTestPublicIPNameForBackendPool, nil)
		if err != nil {
			t.Fatalf("Failed to get public IP address: %v", err)
		}

		err = createLoadBalancerWithBackendPool(ctx, lbClient, subscriptionID, integrationTestResourceGroup, integrationTestBackendPoolLBName, integrationTestLocation, *publicIPResp.ID, integrationTestBackendPoolName)
		if err != nil {
			t.Fatalf("Failed to create load balancer: %v", err)
		}

		log.Printf("Setup completed: Load balancer %s with backend pool %s created", integrationTestBackendPoolLBName, integrationTestBackendPoolName)
		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetBackendAddressPool", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving backend address pool %s from load balancer %s", integrationTestBackendPoolName, integrationTestBackendPoolLBName)

			wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(
				clients.NewLoadBalancerBackendAddressPoolsClient(backendPoolClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestBackendPoolLBName, integrationTestBackendPoolName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
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

			expectedUniqueValue := shared.CompositeLookupKey(integrationTestBackendPoolLBName, integrationTestBackendPoolName)
			if uniqueAttrValue != expectedUniqueValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerBackendAddressPool.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerBackendAddressPool, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved backend address pool %s", integrationTestBackendPoolName)
		})

		t.Run("SearchBackendAddressPools", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching backend address pools in load balancer %s", integrationTestBackendPoolLBName)

			wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(
				clients.NewLoadBalancerBackendAddressPoolsClient(backendPoolClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestBackendPoolLBName, true)
			if err != nil {
				t.Fatalf("Failed to search backend address pools: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one backend address pool, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				expectedValue := shared.CompositeLookupKey(integrationTestBackendPoolLBName, integrationTestBackendPoolName)
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedValue {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find backend pool %s in the search results", integrationTestBackendPoolName)
			}

			log.Printf("Found %d backend address pools in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for backend address pool %s", integrationTestBackendPoolName)

			wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(
				clients.NewLoadBalancerBackendAddressPoolsClient(backendPoolClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestBackendPoolLBName, integrationTestBackendPoolName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("Expected linked query to have a non-empty Type")
				}
				if query.GetQuery() == "" {
					t.Error("Expected linked query to have a non-empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Expected linked query to have a non-empty Scope")
				}
			}

			// Verify parent load balancer link exists
			var hasLoadBalancerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkLoadBalancer.String() {
					hasLoadBalancerLink = true
					if liq.GetQuery().GetQuery() != integrationTestBackendPoolLBName {
						t.Errorf("Expected linked query to load balancer %s, got %s", integrationTestBackendPoolLBName, liq.GetQuery().GetQuery())
					}
					break
				}
			}

			if !hasLoadBalancerLink {
				t.Error("Expected linked query to parent load balancer, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for backend address pool %s", len(linkedQueries), integrationTestBackendPoolName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(
				clients.NewLoadBalancerBackendAddressPoolsClient(backendPoolClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestBackendPoolLBName, integrationTestBackendPoolName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerBackendAddressPool.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerBackendAddressPool, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for backend address pool %s", integrationTestBackendPoolName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestBackendPoolLBName)
		if err != nil {
			t.Fatalf("Failed to delete load balancer: %v", err)
		}

		err = deletePublicIPForBackendPool(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForBackendPool)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		err = deleteVirtualNetworkForBackendPool(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForBackendPool)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

func createVirtualNetworkForBackendPool(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", vnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.3.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: new(integrationTestSubnetNameForBackendPool),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: new("10.3.0.0/24"),
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

func deleteVirtualNetworkForBackendPool(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, vnetName, nil)
	if err != nil {
		log.Printf("Virtual network %s delete failed (may already be deleted): %v", vnetName, err)
		return nil
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network: %w", err)
	}

	log.Printf("Virtual network %s deleted successfully", vnetName)
	return nil
}

func createPublicIPForBackendPool(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, publicIPName, nil)
	if err == nil {
		log.Printf("Public IP address %s already exists, skipping creation", publicIPName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPName, armnetwork.PublicIPAddress{
		Location: new(location),
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: new(armnetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:   new(armnetwork.IPVersionIPv4),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: new(armnetwork.PublicIPAddressSKUNameStandard),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
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

func deletePublicIPForBackendPool(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, publicIPName, nil)
	if err != nil {
		log.Printf("Public IP address %s delete failed (may already be deleted): %v", publicIPName, err)
		return nil
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete public IP address: %w", err)
	}

	log.Printf("Public IP address %s deleted successfully", publicIPName)
	return nil
}

func createLoadBalancerWithBackendPool(ctx context.Context, client *armnetwork.LoadBalancersClient, subscriptionID, resourceGroupName, lbName, location, publicIPID, backendPoolName string) error {
	_, err := client.Get(ctx, resourceGroupName, lbName, nil)
	if err == nil {
		log.Printf("Load balancer %s already exists, skipping creation", lbName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, lbName, armnetwork.LoadBalancer{
		Location: new(location),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: new("frontend-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: new(publicIPID),
						},
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: new(backendPoolName),
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: new("lb-rule"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/frontend-config", subscriptionID, resourceGroupName, lbName)),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", subscriptionID, resourceGroupName, lbName, backendPoolName)),
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

	log.Printf("Load balancer %s with backend pool %s created successfully", lbName, backendPoolName)
	return nil
}
