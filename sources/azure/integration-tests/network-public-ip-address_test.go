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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestPublicIPName     = "ovm-integ-test-public-ip"
	integrationTestNICNameForPIP    = "ovm-integ-test-nic-for-pip"
	integrationTestVNetNameForPIP   = "ovm-integ-test-vnet-for-pip"
	integrationTestSubnetNameForPIP = "default"
)

func TestNetworkPublicIPAddressIntegration(t *testing.T) {
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

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network for the NIC
		err = createVirtualNetworkForPIP(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForPIP, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for NIC creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetNameForPIP, integrationTestSubnetNameForPIP, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create public IP address first (needed for NIC)
		err = createPublicIPAddress(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		// Wait for public IP to be available
		err = waitForPublicIPAvailable(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPName)
		if err != nil {
			t.Fatalf("Failed waiting for public IP to be available: %v", err)
		}

		// Get public IP ID for NIC creation
		publicIPResp, err := publicIPClient.Get(ctx, integrationTestResourceGroup, integrationTestPublicIPName, nil)
		if err != nil {
			t.Fatalf("Failed to get public IP address: %v", err)
		}

		// Create network interface with public IP
		err = createNetworkInterfaceWithPublicIP(ctx, nicClient, integrationTestResourceGroup, integrationTestNICNameForPIP, integrationTestLocation, *subnetResp.ID, *publicIPResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		log.Printf("Setup completed: Public IP address %s and network interface %s created", integrationTestPublicIPName, integrationTestNICNameForPIP)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetPublicIPAddress", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving public IP address %s in subscription %s, resource group %s",
				integrationTestPublicIPName, subscriptionID, integrationTestResourceGroup)

			publicIPWrapper := manual.NewNetworkPublicIPAddress(
				clients.NewPublicIPAddressesClient(publicIPClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := publicIPWrapper.Scopes()[0]

			publicIPAdapter := sources.WrapperToAdapter(publicIPWrapper)
			sdpItem, qErr := publicIPAdapter.Get(ctx, scope, integrationTestPublicIPName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.NetworkPublicIPAddress.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkPublicIPAddress, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != integrationTestPublicIPName {
				t.Errorf("Expected unique attribute value %s, got %s", integrationTestPublicIPName, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved public IP address %s", integrationTestPublicIPName)
		})

		t.Run("ListPublicIPAddresses", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing public IP addresses in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			publicIPWrapper := manual.NewNetworkPublicIPAddress(
				clients.NewPublicIPAddressesClient(publicIPClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := publicIPWrapper.Scopes()[0]

			publicIPAdapter := sources.WrapperToAdapter(publicIPWrapper)
			listable, ok := publicIPAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one public IP address, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == integrationTestPublicIPName {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find public IP address %s in the list results", integrationTestPublicIPName)
			}

			log.Printf("Found %d public IP addresses in list results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for public IP address %s", integrationTestPublicIPName)

			publicIPWrapper := manual.NewNetworkPublicIPAddress(
				clients.NewPublicIPAddressesClient(publicIPClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := publicIPWrapper.Scopes()[0]

			publicIPAdapter := sources.WrapperToAdapter(publicIPWrapper)
			sdpItem, qErr := publicIPAdapter.Get(ctx, scope, integrationTestPublicIPName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (Network Interface should be linked via IPConfiguration)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasNetworkInterfaceLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkNetworkInterface.String() {
					hasNetworkInterfaceLink = true
					if liq.GetQuery().GetQuery() != integrationTestNICNameForPIP {
						t.Errorf("Expected linked query to network interface %s, got %s", integrationTestNICNameForPIP, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}
					// Verify blast propagation
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Error("Expected BlastPropagation to be set for network interface link")
					} else {
						if !bp.GetIn() {
							t.Error("Expected BlastPropagation.In to be true for network interface link")
						}
						if bp.GetOut() {
							t.Error("Expected BlastPropagation.Out to be false for network interface link")
						}
					}
					break
				}
			}

			if !hasNetworkInterfaceLink {
				t.Error("Expected linked query to Network Interface, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for public IP address %s", len(linkedQueries), integrationTestPublicIPName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete network interface first (it must be deleted before public IP can be deleted if associated)
		err := deleteNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICNameForPIP)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		// Delete public IP address
		err = deletePublicIPAddress(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPName)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForPIP(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetNameForPIP)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

// createVirtualNetworkForPIP creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForPIP(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
					Name: ptr.To(integrationTestSubnetNameForPIP),
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

// deleteVirtualNetworkForPIP deletes an Azure virtual network
func deleteVirtualNetworkForPIP(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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

// createPublicIPAddress creates an Azure public IP address (idempotent)
func createPublicIPAddress(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName, location string) error {
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
			PublicIPAddressVersion:   ptr.To(armnetwork.IPVersionIPv4),
			PublicIPAllocationMethod: ptr.To(armnetwork.IPAllocationMethodStatic),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: ptr.To(armnetwork.PublicIPAddressSKUNameStandard),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("network-public-ip-address"),
		},
	}, nil)
	if err != nil {
		// Check if public IP already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Public IP address %s already exists, skipping creation", publicIPName)
			return nil
		}
		return fmt.Errorf("failed to begin creating public IP address: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create public IP address: %w", err)
	}

	log.Printf("Public IP address %s created successfully", publicIPName)
	return nil
}

// waitForPublicIPAvailable waits for a public IP address to be fully available
func waitForPublicIPAvailable(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for public IP address %s to be available via API...", publicIPName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, publicIPName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Public IP address %s not yet available (attempt %d/%d), waiting %v...", publicIPName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking public IP address availability: %w", err)
		}

		// If we can get the public IP and it has an IP address assigned, it's available
		if resp.Properties != nil && resp.Properties.IPAddress != nil && *resp.Properties.IPAddress != "" {
			log.Printf("Public IP address %s is available with IP: %s", publicIPName, *resp.Properties.IPAddress)
			return nil
		}

		// Still provisioning, wait and retry
		log.Printf("Public IP address %s still provisioning (attempt %d/%d), waiting...", publicIPName, attempt, maxAttempts)
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for public IP address %s to be available after %d attempts", publicIPName, maxAttempts)
}

// deletePublicIPAddress deletes an Azure public IP address
func deletePublicIPAddress(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, publicIPName string) error {
	// Check if public IP exists
	_, err := client.Get(ctx, resourceGroupName, publicIPName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Public IP address %s does not exist, skipping deletion", publicIPName)
			return nil
		}
		return fmt.Errorf("error checking public IP address existence: %w", err)
	}

	// Delete the public IP address
	poller, err := client.BeginDelete(ctx, resourceGroupName, publicIPName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting public IP address: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete public IP address: %w", err)
	}

	log.Printf("Public IP address %s deleted successfully", publicIPName)
	return nil
}

// createNetworkInterfaceWithPublicIP creates an Azure network interface with a public IP address (idempotent)
func createNetworkInterfaceWithPublicIP(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName, location, subnetID, publicIPID string) error {
	// Check if NIC already exists
	_, err := client.Get(ctx, resourceGroupName, nicName, nil)
	if err == nil {
		log.Printf("Network interface %s already exists, skipping creation", nicName)
		return nil
	}

	// Create the NIC
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, nicName, armnetwork.Interface{
		Location: ptr.To(location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: ptr.To("ipconfig1"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: ptr.To(subnetID),
						},
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: ptr.To(publicIPID),
						},
						PrivateIPAllocationMethod: ptr.To(armnetwork.IPAllocationMethodDynamic),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("network-public-ip-address"),
		},
	}, nil)
	if err != nil {
		// Check if NIC already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Network interface %s already exists, skipping creation", nicName)
			return nil
		}
		return fmt.Errorf("failed to begin creating network interface: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	log.Printf("Network interface %s created successfully", nicName)
	return nil
}
