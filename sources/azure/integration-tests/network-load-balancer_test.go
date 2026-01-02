package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestLBName            = "ovm-integ-test-lb"
	integrationTestVNetNameForLB     = "ovm-integ-test-vnet-for-lb"
	integrationTestSubnetNameForLB   = "default"
	integrationTestPublicIPNameForLB = "ovm-integ-test-public-ip-for-lb"
)

func TestNetworkLoadBalancerIntegration(t *testing.T) {
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

	lbClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancers client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network for the load balancer
		err = createVirtualNetworkForLB(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForLB, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for load balancer creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetNameForLB, integrationTestSubnetNameForLB, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create public IP address for the load balancer
		err = createPublicIPForLB(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForLB, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		// Get public IP ID
		publicIPResp, err := publicIPClient.Get(ctx, integrationTestResourceGroup, integrationTestPublicIPNameForLB, nil)
		if err != nil {
			t.Fatalf("Failed to get public IP address: %v", err)
		}

		// Create load balancer
		err = createLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestLBName, integrationTestLocation, *subnetResp.ID, *publicIPResp.ID)
		if err != nil {
			t.Fatalf("Failed to create load balancer: %v", err)
		}

		log.Printf("Setup completed: Load balancer %s created", integrationTestLBName)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetLoadBalancer", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving load balancer %s in subscription %s, resource group %s",
				integrationTestLBName, subscriptionID, integrationTestResourceGroup)

			lbWrapper := manual.NewNetworkLoadBalancer(
				clients.NewLoadBalancersClient(lbClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := lbWrapper.Scopes()[0]

			lbAdapter := sources.WrapperToAdapter(lbWrapper)
			sdpItem, qErr := lbAdapter.Get(ctx, scope, integrationTestLBName, true)
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

			if uniqueAttrValue != integrationTestLBName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestLBName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancer.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.NetworkLoadBalancer, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved load balancer %s", integrationTestLBName)
		})

		t.Run("ListLoadBalancers", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing load balancers in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			lbWrapper := manual.NewNetworkLoadBalancer(
				clients.NewLoadBalancersClient(lbClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := lbWrapper.Scopes()[0]

			lbAdapter := sources.WrapperToAdapter(lbWrapper)
			listable, ok := lbAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) == 0 {
				t.Fatalf("Expected at least 1 load balancer, got: %d", len(sdpItems))
			}

			// Find our test load balancer
			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestLBName {
					found = true
					if item.GetType() != azureshared.NetworkLoadBalancer.String() {
						t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancer, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find load balancer %s in list, but didn't", integrationTestLBName)
			}

			log.Printf("Successfully listed %d load balancers", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			lbWrapper := manual.NewNetworkLoadBalancer(
				clients.NewLoadBalancersClient(lbClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := lbWrapper.Scopes()[0]

			lbAdapter := sources.WrapperToAdapter(lbWrapper)
			sdpItem, qErr := lbAdapter.Get(ctx, scope, integrationTestLBName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkLoadBalancer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancer, sdpItem.GetType())
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

			log.Printf("Verified item attributes for load balancer %s", integrationTestLBName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			lbWrapper := manual.NewNetworkLoadBalancer(
				clients.NewLoadBalancersClient(lbClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := lbWrapper.Scopes()[0]

			lbAdapter := sources.WrapperToAdapter(lbWrapper)
			sdpItem, qErr := lbAdapter.Get(ctx, scope, integrationTestLBName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected linked item types
			expectedLinkedTypes := map[string]bool{
				azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(): false,
				azureshared.NetworkPublicIPAddress.String():                    false,
				azureshared.NetworkSubnet.String():                              false,
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true

					// Verify blast propagation based on resource type
					if liq.GetBlastPropagation() == nil {
						t.Errorf("Expected blast propagation to be set for linked type %s", linkedType)
					} else {
						switch linkedType {
						case azureshared.NetworkLoadBalancerFrontendIPConfiguration.String():
							// Child resource: bidirectional dependency
							if liq.GetBlastPropagation().GetIn() != true {
								t.Errorf("Expected FrontendIPConfiguration blast propagation In=true, got false")
							}
							if liq.GetBlastPropagation().GetOut() != true {
								t.Errorf("Expected FrontendIPConfiguration blast propagation Out=true, got false")
							}
						case azureshared.NetworkPublicIPAddress.String():
							// External resource: Public IP affects LB, but LB doesn't affect Public IP
							if liq.GetBlastPropagation().GetIn() != true {
								t.Errorf("Expected PublicIPAddress blast propagation In=true, got false")
							}
							if liq.GetBlastPropagation().GetOut() != false {
								t.Errorf("Expected PublicIPAddress blast propagation Out=false, got true")
							}
						case azureshared.NetworkSubnet.String():
							// External resource: Subnet affects LB, but LB doesn't affect Subnet
							if liq.GetBlastPropagation().GetIn() != true {
								t.Errorf("Expected Subnet blast propagation In=true, got false")
							}
							if liq.GetBlastPropagation().GetOut() != false {
								t.Errorf("Expected Subnet blast propagation Out=false, got true")
							}
						}
					}
				}
			}

			// Verify all expected linked types were found
			for linkedType, found := range expectedLinkedTypes {
				if !found {
					t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
				}
			}

			log.Printf("Verified %d linked item queries for load balancer %s", len(linkedQueries), integrationTestLBName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete load balancer
		err := deleteLoadBalancer(ctx, lbClient, integrationTestResourceGroup, integrationTestLBName)
		if err != nil {
			t.Fatalf("Failed to delete load balancer: %v", err)
		}

		// Delete public IP address
		err = deletePublicIPForLB(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPNameForLB)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForLB(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForLB)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

// createVirtualNetworkForLB creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForLB(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
				AddressPrefixes: []*string{ptr.To("10.2.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: ptr.To(integrationTestSubnetNameForLB),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: ptr.To("10.2.0.0/24"),
					},
				},
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

// deleteVirtualNetworkForLB deletes an Azure virtual network
func deleteVirtualNetworkForLB(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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

// createPublicIPForLB creates an Azure public IP address (idempotent)
func createPublicIPForLB(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName, location string) error {
	// Check if public IP already exists
	_, err := client.Get(ctx, resourceGroupName, publicIPName, nil)
	if err == nil {
		log.Printf("Public IP address %s already exists, skipping creation", publicIPName)
		return nil
	}

	// Create the public IP address
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPName, armnetwork.PublicIPAddress{
		Location: ptr.To(location),
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: ptr.To(armnetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:    ptr.To(armnetwork.IPVersionIPv4),
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

// deletePublicIPForLB deletes an Azure public IP address
func deletePublicIPForLB(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName string) error {
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

// createLoadBalancer creates an Azure load balancer (idempotent)
func createLoadBalancer(ctx context.Context, client *armnetwork.LoadBalancersClient, resourceGroupName, lbName, location, subnetID, publicIPID string) error {
	// Check if load balancer already exists
	_, err := client.Get(ctx, resourceGroupName, lbName, nil)
	if err == nil {
		log.Printf("Load balancer %s already exists, skipping creation", lbName)
		return nil
	}

	// Create the load balancer
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, lbName, armnetwork.LoadBalancer{
		Location: ptr.To(location),
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: ptr.To("frontend-ip-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: ptr.To(publicIPID),
						},
						// Note: Frontend IP configurations must be either public (with PublicIPAddress)
						// or internal (with Subnet), but not both. Since we're using a public IP,
						// we don't include the Subnet here.
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: ptr.To("backend-pool"),
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: ptr.To("lb-rule"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{
							ID: ptr.To(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/frontend-ip-config", os.Getenv("AZURE_SUBSCRIPTION_ID"), resourceGroupName, lbName)),
						},
						BackendAddressPool: &armnetwork.SubResource{
							ID: ptr.To(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/backend-pool", os.Getenv("AZURE_SUBSCRIPTION_ID"), resourceGroupName, lbName)),
						},
						Protocol:           ptr.To(armnetwork.TransportProtocolTCP),
						FrontendPort:       ptr.To[int32](80),
						BackendPort:        ptr.To[int32](80),
						EnableFloatingIP:   ptr.To(false),
						IdleTimeoutInMinutes: ptr.To[int32](4),
					},
				},
			},
		},
		SKU: &armnetwork.LoadBalancerSKU{
			Name: ptr.To(armnetwork.LoadBalancerSKUNameStandard),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
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

// deleteLoadBalancer deletes an Azure load balancer
func deleteLoadBalancer(ctx context.Context, client *armnetwork.LoadBalancersClient, resourceGroupName, lbName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, lbName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Load balancer %s not found, skipping deletion", lbName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting load balancer: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	log.Printf("Load balancer %s deleted successfully", lbName)
	return nil
}

