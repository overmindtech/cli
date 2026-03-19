package integrationtests

import (
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
	integrationTestFrontendIPLBName         = "ovm-integ-test-lb-fip"
	integrationTestFrontendIPPublicIPName   = "ovm-integ-test-pip-fip"
	integrationTestFrontendIPConfigName     = "frontend-ip-config"
	integrationTestFrontendIPVNetName       = "ovm-integ-test-vnet-fip"
	integrationTestFrontendIPSubnetName     = "default"
	integrationTestFrontendIPInternalLBName = "ovm-integ-test-lb-fip-internal"
	integrationTestFrontendIPInternalName   = "frontend-ip-internal"
)

func TestNetworkLoadBalancerFrontendIPConfigurationIntegration(t *testing.T) {
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

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	lbClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancers client: %v", err)
	}

	frontendIPClient, err := armnetwork.NewLoadBalancerFrontendIPConfigurationsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Frontend IP Configurations client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Subnets client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create public IP for public LB
		err = createPublicIPForLB(ctx, publicIPClient, integrationTestResourceGroup, integrationTestFrontendIPPublicIPName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		publicIPResp, err := publicIPClient.Get(ctx, integrationTestResourceGroup, integrationTestFrontendIPPublicIPName, nil)
		if err != nil {
			t.Fatalf("Failed to get public IP address: %v", err)
		}

		// Create public LB with frontend IP config
		err = createPublicLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestFrontendIPLBName, integrationTestLocation, *publicIPResp.ID)
		if err != nil {
			t.Fatalf("Failed to create public load balancer: %v", err)
		}

		// Create VNet + subnet for internal LB
		err = createVirtualNetworkForLB(ctx, vnetClient, integrationTestResourceGroup, integrationTestFrontendIPVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestFrontendIPVNetName, integrationTestFrontendIPSubnetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create internal LB
		err = createInternalLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestFrontendIPInternalLBName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create internal load balancer: %v", err)
		}

		log.Printf("Setup completed for frontend IP configuration integration tests")
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetFrontendIPConfiguration", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(
				clients.NewLoadBalancerFrontendIPConfigurationsClient(frontendIPClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			// The public LB has a frontend IP config named "frontend-ip-config-public"
			query := shared.CompositeLookupKey(integrationTestFrontendIPLBName, "frontend-ip-config-public")
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerFrontendIPConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerFrontendIPConfiguration, sdpItem.GetType())
			}

			expectedUniqueValue := shared.CompositeLookupKey(integrationTestFrontendIPLBName, "frontend-ip-config-public")
			if sdpItem.UniqueAttributeValue() != expectedUniqueValue {
				t.Errorf("Expected unique value %s, got %s", expectedUniqueValue, sdpItem.UniqueAttributeValue())
			}

			log.Printf("Successfully retrieved frontend IP configuration for LB %s", integrationTestFrontendIPLBName)
		})

		t.Run("SearchFrontendIPConfigurations", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(
				clients.NewLoadBalancerFrontendIPConfigurationsClient(frontendIPClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestFrontendIPLBName, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least 1 frontend IP configuration, got: %d", len(sdpItems))
			}

			for _, item := range sdpItems {
				if item.GetType() != azureshared.NetworkLoadBalancerFrontendIPConfiguration.String() {
					t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerFrontendIPConfiguration, item.GetType())
				}
			}

			log.Printf("Successfully searched %d frontend IP configurations for LB %s", len(sdpItems), integrationTestFrontendIPLBName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(
				clients.NewLoadBalancerFrontendIPConfigurationsClient(frontendIPClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			// Verify public frontend IP config links
			t.Run("PublicFrontendIP", func(t *testing.T) {
				query := shared.CompositeLookupKey(integrationTestFrontendIPLBName, "frontend-ip-config-public")
				sdpItem, qErr := adapter.Get(ctx, scope, query, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				linkedQueries := sdpItem.GetLinkedItemQueries()
				for _, liq := range linkedQueries {
					q := liq.GetQuery()
					if q.GetType() == "" {
						t.Error("Linked item query has empty Type")
					}
					if q.GetQuery() == "" {
						t.Errorf("Linked item query of type %s has empty Query", q.GetType())
					}
					if q.GetScope() == "" {
						t.Errorf("Linked item query of type %s has empty Scope", q.GetType())
					}
				}

				expectedTypes := map[string]bool{
					azureshared.NetworkLoadBalancer.String():    false,
					azureshared.NetworkPublicIPAddress.String(): false,
				}

				for _, liq := range linkedQueries {
					if _, exists := expectedTypes[liq.GetQuery().GetType()]; exists {
						expectedTypes[liq.GetQuery().GetType()] = true
					}
				}

				for linkedType, found := range expectedTypes {
					if !found {
						t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
					}
				}
			})

			// Verify internal frontend IP config links
			t.Run("InternalFrontendIP", func(t *testing.T) {
				query := shared.CompositeLookupKey(integrationTestFrontendIPInternalLBName, "frontend-ip-config-internal")
				sdpItem, qErr := adapter.Get(ctx, scope, query, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				linkedQueries := sdpItem.GetLinkedItemQueries()

				expectedTypes := map[string]bool{
					azureshared.NetworkLoadBalancer.String(): false,
					azureshared.NetworkSubnet.String():       false,
					"ip":                                     false,
				}

				for _, liq := range linkedQueries {
					if _, exists := expectedTypes[liq.GetQuery().GetType()]; exists {
						expectedTypes[liq.GetQuery().GetType()] = true
					}
				}

				for linkedType, found := range expectedTypes {
					if !found {
						t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
					}
				}
			})
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(
				clients.NewLoadBalancerFrontendIPConfigurationsClient(frontendIPClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestFrontendIPLBName, "frontend-ip-config-public")
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerFrontendIPConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerFrontendIPConfiguration, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete public LB
		err := deleteLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestFrontendIPLBName)
		if err != nil {
			t.Fatalf("Failed to delete public load balancer: %v", err)
		}

		// Delete internal LB
		err = deleteLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestFrontendIPInternalLBName)
		if err != nil {
			t.Fatalf("Failed to delete internal load balancer: %v", err)
		}

		// Delete public IP
		err = deletePublicIPForLB(ctx, publicIPClient, integrationTestResourceGroup, integrationTestFrontendIPPublicIPName)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		// Delete VNet
		err = deleteVirtualNetworkForLB(ctx, vnetClient, integrationTestResourceGroup, integrationTestFrontendIPVNetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}
