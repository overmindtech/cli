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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
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
	integrationTestVMName     = "ovm-integ-test-vm"
	integrationTestNICName    = "ovm-integ-test-nic"
	integrationTestVNetName   = "ovm-integ-test-vnet"
	integrationTestSubnetName = "default"

	defaultMaxPollAttempts = 20
	defaultPollInterval    = 10 * time.Second
)

func TestComputeVirtualMachineIntegration(t *testing.T) {
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
	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Machines client: %v", err)
	}

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

		// Create virtual network
		err = createVirtualNetwork(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for NIC creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetName, integrationTestSubnetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create network interface
		err = createNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		// Get NIC ID for VM creation
		nicResp, err := nicClient.Get(ctx, integrationTestResourceGroup, integrationTestNICName, nil)
		if err != nil {
			t.Fatalf("Failed to get network interface: %v", err)
		}

		// Create virtual machine
		err = createVirtualMachine(ctx, vmClient, integrationTestResourceGroup, integrationTestVMName, integrationTestLocation, *nicResp.ID)
		if err != nil {
			t.Fatalf("Failed to create virtual machine: %v", err)
		}

		// Wait for VM to be fully available via the API
		err = waitForVMAvailable(ctx, vmClient, integrationTestResourceGroup, integrationTestVMName)
		if err != nil {
			t.Fatalf("Failed waiting for VM to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetVirtualMachine", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving virtual machine %s in subscription %s, resource group %s",
				integrationTestVMName, subscriptionID, integrationTestResourceGroup)

			vmWrapper := manual.NewComputeVirtualMachine(
				clients.NewVirtualMachinesClient(vmClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmWrapper.Scopes()[0]

			vmAdapter := sources.WrapperToAdapter(vmWrapper)
			sdpItem, qErr := vmAdapter.Get(ctx, scope, integrationTestVMName, true)
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

			if uniqueAttrValue != integrationTestVMName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestVMName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved virtual machine %s", integrationTestVMName)
		})

		t.Run("ListVirtualMachines", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing virtual machines in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			vmWrapper := manual.NewComputeVirtualMachine(
				clients.NewVirtualMachinesClient(vmClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmWrapper.Scopes()[0]

			vmAdapter := sources.WrapperToAdapter(vmWrapper)

			// Check if adapter supports listing
			listable, ok := vmAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list virtual machines: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one virtual machine, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestVMName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find VM %s in the list of virtual machines", integrationTestVMName)
			}

			log.Printf("Found %d virtual machines in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for virtual machine %s", integrationTestVMName)

			vmWrapper := manual.NewComputeVirtualMachine(
				clients.NewVirtualMachinesClient(vmClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmWrapper.Scopes()[0]

			vmAdapter := sources.WrapperToAdapter(vmWrapper)
			sdpItem, qErr := vmAdapter.Get(ctx, scope, integrationTestVMName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (OS disk, NIC, run commands should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasDiskLink, hasNICLink, hasRunCommandLink bool
			for _, liq := range linkedQueries {
				switch liq.GetQuery().GetType() {
				case azureshared.ComputeDisk.String():
					hasDiskLink = true
				case azureshared.NetworkNetworkInterface.String():
					hasNICLink = true
				case azureshared.ComputeVirtualMachineRunCommand.String():
					hasRunCommandLink = true
					// Verify run command link properties
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected run command link method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetQuery() != integrationTestVMName {
						t.Errorf("Expected run command link query to be %s, got %s", integrationTestVMName, liq.GetQuery().GetQuery())
					}
					// Verify blast propagation (In: false, Out: true)
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected run command blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected run command blast propagation Out=true, got false")
					}
				case azureshared.ComputeVirtualMachineExtension.String():
					// Extensions may or may not be present depending on VM setup
					// Verify extension link properties if present
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected extension link method to be GET, got %s", liq.GetQuery().GetMethod())
					}
					// Verify blast propagation (In: false, Out: true)
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected extension blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected extension blast propagation Out=true, got false")
					}
				}
			}

			if !hasDiskLink {
				t.Error("Expected linked query to OS disk, but didn't find one")
			}

			if !hasNICLink {
				t.Error("Expected linked query to network interface, but didn't find one")
			}

			// Run commands link should always be present (even if no run commands exist)
			if !hasRunCommandLink {
				t.Error("Expected linked query to run commands, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for VM %s", len(linkedQueries), integrationTestVMName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete VM first (it must be deleted before NIC can be deleted)
		err := deleteVirtualMachine(ctx, vmClient, integrationTestResourceGroup, integrationTestVMName)
		if err != nil {
			t.Fatalf("Failed to delete virtual machine: %v", err)
		}

		// Delete NIC
		err = deleteNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICName)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetwork(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
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

// createVirtualNetwork creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetwork(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
				AddressPrefixes: []*string{ptr.To("10.0.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: ptr.To(integrationTestSubnetName),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: ptr.To("10.0.0.0/24"),
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

// createNetworkInterface creates an Azure network interface (idempotent)
func createNetworkInterface(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName, location, subnetID string) error {
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
						PrivateIPAllocationMethod: ptr.To(armnetwork.IPAllocationMethodDynamic),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating network interface: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	log.Printf("Network interface %s created successfully", nicName)
	return nil
}

// createVirtualMachine creates an Azure virtual machine (idempotent)
func createVirtualMachine(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName, location, nicID string) error {
	// Check if VM already exists
	existingVM, err := client.Get(ctx, resourceGroupName, vmName, nil)
	if err == nil {
		// VM exists, check its state
		if existingVM.Properties != nil && existingVM.Properties.ProvisioningState != nil {
			state := *existingVM.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Virtual machine %s already exists with state %s, skipping creation", vmName, state)
				return nil
			}
			log.Printf("Virtual machine %s exists but in state %s, will wait for it", vmName, state)
		} else {
			log.Printf("Virtual machine %s already exists, skipping creation", vmName)
			return nil
		}
	}

	// Create the VM
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vmName, armcompute.VirtualMachine{
		Location: ptr.To(location),
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				// Use Standard_D2ps_v5 - ARM-based VM with good availability in westus2
				VMSize: ptr.To(armcompute.VirtualMachineSizeTypes("Standard_D2ps_v5")),
			},
			StorageProfile: &armcompute.StorageProfile{
				ImageReference: &armcompute.ImageReference{
					Publisher: ptr.To("Canonical"),
					Offer:     ptr.To("0001-com-ubuntu-server-jammy"),
					SKU:       ptr.To("22_04-lts-arm64"), // ARM64 image for ARM-based VM
					Version:   ptr.To("latest"),
				},
				OSDisk: &armcompute.OSDisk{
					Name:         ptr.To(fmt.Sprintf("%s-osdisk", vmName)),
					CreateOption: ptr.To(armcompute.DiskCreateOptionTypesFromImage),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						StorageAccountType: ptr.To(armcompute.StorageAccountTypesStandardLRS),
					},
					DeleteOption: ptr.To(armcompute.DiskDeleteOptionTypesDelete),
				},
			},
			OSProfile: &armcompute.OSProfile{
				ComputerName:  ptr.To(vmName),
				AdminUsername: ptr.To("azureuser"),
				// Use password authentication for integration tests (simpler than SSH keys)
				AdminPassword: ptr.To("OvmIntegTest2024!"),
				LinuxConfiguration: &armcompute.LinuxConfiguration{
					DisablePasswordAuthentication: ptr.To(false),
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: ptr.To(nicID),
						Properties: &armcompute.NetworkInterfaceReferenceProperties{
							Primary: ptr.To(true),
						},
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-virtual-machine"),
		},
	}, nil)
	if err != nil {
		// Check if VM already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Virtual machine %s already exists (conflict), skipping creation", vmName)
			return nil
		}
		return fmt.Errorf("failed to begin creating virtual machine: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual machine: %w", err)
	}

	// Verify the VM was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("VM created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("VM provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Virtual machine %s created successfully with provisioning state: %s", vmName, provisioningState)
	return nil
}

// waitForVMAvailable polls until the VM is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the VM is queryable
func waitForVMAvailable(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName string) error {
	maxAttempts := defaultMaxPollAttempts
	pollInterval := defaultPollInterval

	log.Printf("Waiting for VM %s to be available via API...", vmName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, vmName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("VM %s not yet available (attempt %d/%d), waiting %v...", vmName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking VM availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("VM %s is available with provisioning state: %s", vmName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("VM provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("VM %s provisioning state: %s (attempt %d/%d), waiting...", vmName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// VM exists but no provisioning state - consider it available
		log.Printf("VM %s is available", vmName)
		return nil
	}

	return fmt.Errorf("timeout waiting for VM %s to be available after %d attempts", vmName, maxAttempts)
}

// deleteVirtualMachine deletes an Azure virtual machine
func deleteVirtualMachine(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName string) error {
	// Use forceDeletion to speed up cleanup
	poller, err := client.BeginDelete(ctx, resourceGroupName, vmName, &armcompute.VirtualMachinesClientBeginDeleteOptions{
		ForceDeletion: ptr.To(true),
	})
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual machine %s not found, skipping deletion", vmName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting virtual machine: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual machine: %w", err)
	}

	log.Printf("Virtual machine %s deleted successfully", vmName)

	// Wait a bit to allow Azure to release associated resources
	log.Printf("Waiting 30 seconds for Azure to release associated resources...")
	time.Sleep(30 * time.Second)

	return nil
}

// deleteNetworkInterface deletes an Azure network interface with retry logic
// Azure reserves NICs for 180 seconds after VM deletion, so we may need to retry
func deleteNetworkInterface(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName string) error {
	maxRetries := 4
	retryDelay := 60 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		poller, err := client.BeginDelete(ctx, resourceGroupName, nicName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) {
				if respErr.StatusCode == http.StatusNotFound {
					log.Printf("Network interface %s not found, skipping deletion", nicName)
					return nil
				}
				// Handle NicReservedForAnotherVm error - retry after delay
				if respErr.ErrorCode == "NicReservedForAnotherVm" && attempt < maxRetries {
					log.Printf("NIC %s is reserved, waiting %v before retry (attempt %d/%d)", nicName, retryDelay, attempt, maxRetries)
					time.Sleep(retryDelay)
					continue
				}
			}
			return fmt.Errorf("failed to begin deleting network interface: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to delete network interface: %w", err)
		}

		log.Printf("Network interface %s deleted successfully", nicName)
		return nil
	}

	return fmt.Errorf("failed to delete network interface %s after %d attempts", nicName, maxRetries)
}

// deleteVirtualNetwork deletes an Azure virtual network
func deleteVirtualNetwork(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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
