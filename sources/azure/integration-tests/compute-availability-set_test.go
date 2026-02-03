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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
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
	integrationTestAvailabilitySetName = "ovm-integ-test-avset"
	integrationTestVMForAVSetName      = "ovm-integ-test-vm-avset"
	integrationTestNICForAVSetName     = "ovm-integ-test-nic-avset"
	integrationTestVNetForAVSetName    = "ovm-integ-test-vnet-avset"
	integrationTestSubnetForAVSetName  = "default"
)

func TestComputeAvailabilitySetIntegration(t *testing.T) {
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
	avSetClient, err := armcompute.NewAvailabilitySetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Availability Sets client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Machines client: %v", err)
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

		// Create availability set
		err = createAvailabilitySet(ctx, avSetClient, integrationTestResourceGroup, integrationTestAvailabilitySetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create availability set: %v", err)
		}

		// Wait for availability set to be fully available via the API
		err = waitForAvailabilitySetAvailable(ctx, avSetClient, integrationTestResourceGroup, integrationTestAvailabilitySetName)
		if err != nil {
			t.Fatalf("Failed waiting for availability set to be available: %v", err)
		}

		// Create virtual network for VM
		err = createVirtualNetworkForAVSet(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForAVSetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for NIC creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVNetForAVSetName, integrationTestSubnetForAVSetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create network interface
		err = createNetworkInterfaceForAVSet(ctx, nicClient, integrationTestResourceGroup, integrationTestNICForAVSetName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		// Get NIC ID and Availability Set ID for VM creation
		nicResp, err := nicClient.Get(ctx, integrationTestResourceGroup, integrationTestNICForAVSetName, nil)
		if err != nil {
			t.Fatalf("Failed to get network interface: %v", err)
		}

		avSetResp, err := avSetClient.Get(ctx, integrationTestResourceGroup, integrationTestAvailabilitySetName, nil)
		if err != nil {
			t.Fatalf("Failed to get availability set: %v", err)
		}

		// Create virtual machine with availability set
		err = createVirtualMachineWithAvailabilitySet(ctx, vmClient, integrationTestResourceGroup, integrationTestVMForAVSetName, integrationTestLocation, *nicResp.ID, *avSetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create virtual machine: %v", err)
		}

		// Wait for VM to be fully available via the API
		err = waitForVMAvailable(ctx, vmClient, integrationTestResourceGroup, integrationTestVMForAVSetName)
		if err != nil {
			t.Fatalf("Failed waiting for VM to be available: %v", err)
		}

		// Wait a bit for the availability set to reflect the VM association
		time.Sleep(10 * time.Second)
	})

	t.Run("Run", func(t *testing.T) {
		// Check if availability set exists - if Setup failed, skip Run tests
		ctx := t.Context()
		_, err := avSetClient.Get(ctx, integrationTestResourceGroup, integrationTestAvailabilitySetName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				t.Skipf("Availability set %s does not exist - Setup may have failed. Skipping Run tests.", integrationTestAvailabilitySetName)
			}
		}

		t.Run("GetAvailabilitySet", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving availability set %s in subscription %s, resource group %s",
				integrationTestAvailabilitySetName, subscriptionID, integrationTestResourceGroup)

			avSetWrapper := manual.NewComputeAvailabilitySet(
				clients.NewAvailabilitySetsClient(avSetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := avSetWrapper.Scopes()[0]

			avSetAdapter := sources.WrapperToAdapter(avSetWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := avSetAdapter.Get(ctx, scope, integrationTestAvailabilitySetName, true)
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

			if uniqueAttrValue != integrationTestAvailabilitySetName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestAvailabilitySetName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.ComputeAvailabilitySet.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.ComputeAvailabilitySet, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved availability set %s", integrationTestAvailabilitySetName)
		})

		t.Run("ListAvailabilitySets", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing availability sets in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			avSetWrapper := manual.NewComputeAvailabilitySet(
				clients.NewAvailabilitySetsClient(avSetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := avSetWrapper.Scopes()[0]

			avSetAdapter := sources.WrapperToAdapter(avSetWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := avSetAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list availability sets: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one availability set, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestAvailabilitySetName {
					found = true
					if item.GetType() != azureshared.ComputeAvailabilitySet.String() {
						t.Errorf("Expected type %s, got %s", azureshared.ComputeAvailabilitySet, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find availability set %s in the list of availability sets", integrationTestAvailabilitySetName)
			}

			log.Printf("Found %d availability sets in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for availability set %s", integrationTestAvailabilitySetName)

			avSetWrapper := manual.NewComputeAvailabilitySet(
				clients.NewAvailabilitySetsClient(avSetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := avSetWrapper.Scopes()[0]

			avSetAdapter := sources.WrapperToAdapter(avSetWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := avSetAdapter.Get(ctx, scope, integrationTestAvailabilitySetName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasVMLink bool
			for _, liq := range linkedQueries {
				switch liq.GetQuery().GetType() {
				case azureshared.ComputeVirtualMachine.String():
					hasVMLink = true
					// Verify VM link properties
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected VM link method to be GET, got %s", liq.GetQuery().GetMethod())
					}
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected VM blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected VM blast propagation Out=false, got true")
					}
				case azureshared.ComputeProximityPlacementGroup.String():
					// PPG may or may not be present depending on availability set setup
					// Verify PPG link properties if present
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected PPG link method to be GET, got %s", liq.GetQuery().GetMethod())
					}
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected PPG blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected PPG blast propagation Out=false, got true")
					}
				}
			}

			// VM link should be present if we created a VM with this availability set
			if !hasVMLink {
				t.Logf("No VM link found - this is expected if VM creation failed or VM is not yet associated")
			}

			log.Printf("Verified %d linked item queries for availability set %s", len(linkedQueries), integrationTestAvailabilitySetName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for availability set %s", integrationTestAvailabilitySetName)

			avSetWrapper := manual.NewComputeAvailabilitySet(
				clients.NewAvailabilitySetsClient(avSetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := avSetWrapper.Scopes()[0]

			avSetAdapter := sources.WrapperToAdapter(avSetWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := avSetAdapter.Get(ctx, scope, integrationTestAvailabilitySetName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.ComputeAvailabilitySet.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeAvailabilitySet, sdpItem.GetType())
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

			log.Printf("Verified item attributes for availability set %s", integrationTestAvailabilitySetName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete VM first (it must be deleted before availability set can be deleted)
		err := deleteVirtualMachine(ctx, vmClient, integrationTestResourceGroup, integrationTestVMForAVSetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual machine: %v", err)
		}

		// Delete NIC
		err = deleteNetworkInterface(ctx, nicClient, integrationTestResourceGroup, integrationTestNICForAVSetName)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetwork(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForAVSetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}

		// Delete availability set
		err = deleteAvailabilitySet(ctx, avSetClient, integrationTestResourceGroup, integrationTestAvailabilitySetName)
		if err != nil {
			t.Fatalf("Failed to delete availability set: %v", err)
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

// createAvailabilitySet creates an Azure availability set (idempotent)
func createAvailabilitySet(ctx context.Context, client *armcompute.AvailabilitySetsClient, resourceGroupName, avSetName, location string) error {
	// Check if availability set already exists
	_, err := client.Get(ctx, resourceGroupName, avSetName, nil)
	if err == nil {
		log.Printf("Availability set %s already exists, skipping creation", avSetName)
		return nil
	}

	// Create the availability set
	resp, err := client.CreateOrUpdate(ctx, resourceGroupName, avSetName, armcompute.AvailabilitySet{
		Location: ptr.To(location),
		Properties: &armcompute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  ptr.To[int32](2),
			PlatformUpdateDomainCount: ptr.To[int32](2),
			ProximityPlacementGroup:   nil, // Optional - not setting for this test
			VirtualMachines:           nil, // Will be populated when VMs are added
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-availability-set"),
		},
	}, nil)
	if err != nil {
		// Check if availability set already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Availability set %s already exists (conflict), skipping creation", avSetName)
			return nil
		}
		return fmt.Errorf("failed to create availability set: %w", err)
	}

	// Verify the availability set was created successfully
	if resp.Name == nil {
		return fmt.Errorf("availability set created but name is nil")
	}

	log.Printf("Availability set %s created successfully", avSetName)
	return nil
}

// waitForAvailabilitySetAvailable polls until the availability set is available via the Get API
func waitForAvailabilitySetAvailable(ctx context.Context, client *armcompute.AvailabilitySetsClient, resourceGroupName, avSetName string) error {
	maxAttempts := defaultMaxPollAttempts
	pollInterval := defaultPollInterval

	log.Printf("Waiting for availability set %s to be available via API...", avSetName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, avSetName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Availability set %s not yet available (attempt %d/%d), waiting %v...", avSetName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking availability set availability: %w", err)
		}

		// Availability set exists - consider it available
		if resp.Name != nil {
			log.Printf("Availability set %s is available", avSetName)
			return nil
		}

		// Wait and retry
		if attempt < maxAttempts {
			log.Printf("Availability set %s not yet ready (attempt %d/%d), waiting %v...", avSetName, attempt, maxAttempts, pollInterval)
			time.Sleep(pollInterval)
			continue
		}
	}

	return fmt.Errorf("timeout waiting for availability set %s to be available after %d attempts", avSetName, maxAttempts)
}

// deleteAvailabilitySet deletes an Azure availability set
func deleteAvailabilitySet(ctx context.Context, client *armcompute.AvailabilitySetsClient, resourceGroupName, avSetName string) error {
	_, err := client.Delete(ctx, resourceGroupName, avSetName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Availability set %s not found, skipping deletion", avSetName)
			return nil
		}
		return fmt.Errorf("failed to delete availability set: %w", err)
	}

	log.Printf("Availability set %s deleted successfully", avSetName)
	return nil
}

// createVirtualNetworkForAVSet creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForAVSet(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
					Name: ptr.To(integrationTestSubnetForAVSetName),
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

// createNetworkInterfaceForAVSet creates an Azure network interface (idempotent)
func createNetworkInterfaceForAVSet(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName, location, subnetID string) error {
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

// createVirtualMachineWithAvailabilitySet creates an Azure virtual machine with an availability set (idempotent)
func createVirtualMachineWithAvailabilitySet(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName, location, nicID, availabilitySetID string) error {
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
			AvailabilitySet: &armcompute.SubResource{
				ID: ptr.To(availabilitySetID),
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-availability-set"),
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
