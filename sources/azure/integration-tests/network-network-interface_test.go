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
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestNICNameForTest   = "ovm-integ-test-nic-standalone"
	integrationTestVNetNameForNIC   = "ovm-integ-test-vnet-for-nic"
	integrationTestSubnetNameForNIC = "default"
)

func TestNetworkNetworkInterfaceIntegration(t *testing.T) {
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

	nicClient, err := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Network Interfaces client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network for the NIC
		err = createVirtualNetworkForNIC(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForNIC, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for NIC creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetNameForNIC, integrationTestSubnetNameForNIC, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create network interface
		err = createNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICNameForTest, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		log.Printf("Setup completed: Network interface %s created", integrationTestNICNameForTest)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetNetworkInterface", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving network interface %s in subscription %s, resource group %s",
				integrationTestNICNameForTest, subscriptionID, integrationTestResourceGroup)

			nicWrapper := manual.NewNetworkNetworkInterface(
				clients.NewNetworkInterfacesClient(nicClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := nicWrapper.Scopes()[0]

			nicAdapter := sources.WrapperToAdapter(nicWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := nicAdapter.Get(ctx, scope, integrationTestNICNameForTest, true)
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

			if uniqueAttrValue != integrationTestNICNameForTest {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestNICNameForTest, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved network interface %s", integrationTestNICNameForTest)
		})

		t.Run("ListNetworkInterfaces", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing network interfaces in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			nicWrapper := manual.NewNetworkNetworkInterface(
				clients.NewNetworkInterfacesClient(nicClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := nicWrapper.Scopes()[0]

			nicAdapter := sources.WrapperToAdapter(nicWrapper, sdpcache.NewNoOpCache())
			listable, ok := nicAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) == 0 {
				t.Fatalf("Expected at least 1 network interface, got: %d", len(sdpItems))
			}

			// Find our test NIC
			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestNICNameForTest {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find network interface %s in list, but didn't", integrationTestNICNameForTest)
			}

			log.Printf("Successfully listed %d network interfaces", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			nicWrapper := manual.NewNetworkNetworkInterface(
				clients.NewNetworkInterfacesClient(nicClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := nicWrapper.Scopes()[0]

			nicAdapter := sources.WrapperToAdapter(nicWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := nicAdapter.Get(ctx, scope, integrationTestNICNameForTest, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkNetworkInterface.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkInterface, sdpItem.GetType())
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

			// Verify linked item queries exist (IP configuration link should always be present)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			hasIPConfigLink := false
			for _, liq := range linkedQueries {
				switch liq.GetQuery().GetType() {
				case azureshared.NetworkNetworkInterfaceIPConfiguration.String():
					hasIPConfigLink = true
					// Verify blast propagation (In: false, Out: true)
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected IP config blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected IP config blast propagation Out=true, got false")
					}
				case azureshared.ComputeVirtualMachine.String():
					// VM link may or may not be present depending on whether NIC is attached
					// Verify blast propagation if present (In: false, Out: true)
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected VM blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected VM blast propagation Out=true, got false")
					}
				case azureshared.NetworkNetworkSecurityGroup.String():
					// NSG link may or may not be present
					// Verify blast propagation if present (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected NSG blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected NSG blast propagation Out=false, got true")
					}
				}
			}

			// IP configuration link should always be present
			if !hasIPConfigLink {
				t.Error("Expected linked query to IP configuration, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for NIC %s", len(linkedQueries), integrationTestNICNameForTest)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete network interface
		err := deleteNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICNameForTest)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForNIC(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForNIC)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

// createVirtualNetworkForNIC creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForNIC(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
				AddressPrefixes: []*string{ptr.To("10.1.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: ptr.To(integrationTestSubnetNameForNIC),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: ptr.To("10.1.0.0/24"),
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

// deleteVirtualNetworkForNIC deletes an Azure virtual network
func deleteVirtualNetworkForNIC(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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
