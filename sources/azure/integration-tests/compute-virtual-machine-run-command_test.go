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
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

const (
	integrationTestRunCommandVMName     = "ovm-integ-test-rc-vm"
	integrationTestRunCommandNICName    = "ovm-integ-test-rc-nic"
	integrationTestRunCommandVNetName   = "ovm-integ-test-rc-vnet"
	integrationTestRunCommandSubnetName = "default"
	integrationTestRunCommandName       = "ovm-integ-test-run-command"
)

func TestComputeVirtualMachineRunCommandIntegration(t *testing.T) {
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

	runCommandClient, err := armcompute.NewVirtualMachineRunCommandsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Machine Run Commands client: %v", err)
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
		err = createVirtualNetworkForRunCommand(ctx, vnetClient, integrationTestResourceGroup, integrationTestRunCommandVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for NIC creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestRunCommandVNetName, integrationTestRunCommandSubnetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create network interface
		err = createNetworkInterfaceForRunCommand(ctx, nicClient, integrationTestResourceGroup, integrationTestRunCommandNICName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		// Get NIC ID for VM creation
		nicResp, err := nicClient.Get(ctx, integrationTestResourceGroup, integrationTestRunCommandNICName, nil)
		if err != nil {
			t.Fatalf("Failed to get network interface: %v", err)
		}

		// Create virtual machine
		err = createVirtualMachineForRunCommand(ctx, vmClient, integrationTestResourceGroup, integrationTestRunCommandVMName, integrationTestLocation, *nicResp.ID)
		if err != nil {
			t.Fatalf("Failed to create virtual machine: %v", err)
		}

		// Wait for VM to be fully available via the API
		err = waitForVMAvailableForRunCommand(ctx, vmClient, integrationTestResourceGroup, integrationTestRunCommandVMName)
		if err != nil {
			t.Fatalf("Failed waiting for VM to be available: %v", err)
		}

		// Create run command
		err = createVirtualMachineRunCommand(ctx, runCommandClient, integrationTestResourceGroup, integrationTestRunCommandVMName, integrationTestRunCommandName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual machine run command: %v", err)
		}

		// Wait for run command to be available
		err = waitForRunCommandAvailable(ctx, runCommandClient, integrationTestResourceGroup, integrationTestRunCommandVMName, integrationTestRunCommandName)
		if err != nil {
			t.Fatalf("Failed waiting for run command to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetVirtualMachineRunCommand", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving virtual machine run command %s for VM %s in subscription %s, resource group %s",
				integrationTestRunCommandName, integrationTestRunCommandVMName, subscriptionID, integrationTestResourceGroup)

			runCommandWrapper := manual.NewComputeVirtualMachineRunCommand(
				clients.NewVirtualMachineRunCommandsClient(runCommandClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := runCommandWrapper.Scopes()[0]

			runCommandAdapter := sources.WrapperToAdapter(runCommandWrapper, sdpcache.NewNoOpCache())
			// Get requires virtualMachineName and runCommandName as query parts
			query := integrationTestRunCommandVMName + shared.QuerySeparator + integrationTestRunCommandName
			sdpItem, qErr := runCommandAdapter.Get(ctx, scope, query, true)
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

			expectedUniqueAttr := shared.CompositeLookupKey(integrationTestRunCommandVMName, integrationTestRunCommandName)
			if uniqueAttrValue != expectedUniqueAttr {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedUniqueAttr, uniqueAttrValue)
			}

			// Verify the run command name attribute
			nameAttr, err := sdpItem.GetAttributes().Get("name")
			if err != nil {
				t.Fatalf("Failed to get name attribute: %v", err)
			}
			if nameAttr != integrationTestRunCommandName {
				t.Fatalf("Expected name attribute to be %s, got %s", integrationTestRunCommandName, nameAttr)
			}

			log.Printf("Successfully retrieved virtual machine run command %s", integrationTestRunCommandName)
		})

		t.Run("SearchVirtualMachineRunCommands", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching virtual machine run commands for VM %s", integrationTestRunCommandVMName)

			runCommandWrapper := manual.NewComputeVirtualMachineRunCommand(
				clients.NewVirtualMachineRunCommandsClient(runCommandClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := runCommandWrapper.Scopes()[0]

			runCommandAdapter := sources.WrapperToAdapter(runCommandWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := runCommandAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestRunCommandVMName, true)
			if err != nil {
				t.Fatalf("Failed to search virtual machine run commands: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one virtual machine run command, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				uniqueAttrValue, err := item.GetAttributes().Get(uniqueAttrKey)
				if err != nil {
					continue
				}
				expectedUniqueAttr := shared.CompositeLookupKey(integrationTestRunCommandVMName, integrationTestRunCommandName)
				if uniqueAttrValue == expectedUniqueAttr {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find run command %s in the search results", integrationTestRunCommandName)
			}

			log.Printf("Found %d virtual machine run commands in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for virtual machine run command %s", integrationTestRunCommandName)

			runCommandWrapper := manual.NewComputeVirtualMachineRunCommand(
				clients.NewVirtualMachineRunCommandsClient(runCommandClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := runCommandWrapper.Scopes()[0]

			runCommandAdapter := sources.WrapperToAdapter(runCommandWrapper, sdpcache.NewNoOpCache())
			query := integrationTestRunCommandVMName + shared.QuerySeparator + integrationTestRunCommandName
			sdpItem, qErr := runCommandAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (VM should be linked)
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
					if liq.GetQuery().GetQuery() != integrationTestRunCommandVMName {
						t.Errorf("Expected VM link query to be %s, got %s", integrationTestRunCommandVMName, liq.GetQuery().GetQuery())
					}
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected VM blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected VM blast propagation Out=false, got true")
					}
				case azureshared.StorageAccount.String():
					// Storage account links may be present if outputBlobUri, errorBlobUri, or scriptUri are set
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected Storage Account blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected Storage Account blast propagation Out=false, got true")
					}
				case azureshared.StorageBlobContainer.String():
					// Blob container links may be present if outputBlobUri, errorBlobUri, or scriptUri are set
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected Blob Container blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected Blob Container blast propagation Out=false, got true")
					}
				case stdlib.NetworkHTTP.String():
					// HTTP links may be present if scriptUri is HTTP/HTTPS
					// Verify blast propagation (In: true, Out: true)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected HTTP blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected HTTP blast propagation Out=true, got false")
					}
				case stdlib.NetworkDNS.String():
					// DNS links may be present if scriptUri contains a DNS name
					// Verify blast propagation (In: true, Out: true)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected DNS blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected DNS blast propagation Out=true, got false")
					}
				}
			}

			if !hasVMLink {
				t.Error("Expected linked query to virtual machine, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for run command %s", len(linkedQueries), integrationTestRunCommandName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete run command first
		err := deleteVirtualMachineRunCommand(ctx, runCommandClient, integrationTestResourceGroup, integrationTestRunCommandVMName, integrationTestRunCommandName)
		if err != nil {
			t.Fatalf("Failed to delete virtual machine run command: %v", err)
		}

		// Delete VM (it must be deleted before NIC can be deleted)
		err = deleteVirtualMachineForRunCommand(ctx, vmClient, integrationTestResourceGroup, integrationTestRunCommandVMName)
		if err != nil {
			t.Fatalf("Failed to delete virtual machine: %v", err)
		}

		// Delete NIC
		err = deleteNetworkInterfaceForRunCommand(ctx, nicClient, integrationTestResourceGroup, integrationTestRunCommandNICName)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForRunCommand(ctx, vnetClient, integrationTestResourceGroup, integrationTestRunCommandVNetName)
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

// createVirtualNetworkForRunCommand creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForRunCommand(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
					Name: ptr.To(integrationTestRunCommandSubnetName),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: ptr.To("10.1.0.0/24"),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-virtual-machine-run-command"),
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

// createNetworkInterfaceForRunCommand creates an Azure network interface (idempotent)
func createNetworkInterfaceForRunCommand(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName, location, subnetID string) error {
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
			"test":    ptr.To("compute-virtual-machine-run-command"),
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

// createVirtualMachineForRunCommand creates an Azure virtual machine (idempotent)
func createVirtualMachineForRunCommand(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName, location, nicID string) error {
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
			"test":    ptr.To("compute-virtual-machine-run-command"),
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

// waitForVMAvailableForRunCommand polls until the VM is available via the Get API
func waitForVMAvailableForRunCommand(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName string) error {
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

// createVirtualMachineRunCommand creates an Azure virtual machine run command (idempotent)
func createVirtualMachineRunCommand(ctx context.Context, client *armcompute.VirtualMachineRunCommandsClient, resourceGroupName, vmName, runCommandName, location string) error {
	// Check if run command already exists
	_, err := client.GetByVirtualMachine(ctx, resourceGroupName, vmName, runCommandName, nil)
	if err == nil {
		log.Printf("Virtual machine run command %s already exists, skipping creation", runCommandName)
		return nil
	}

	// Create the run command with a simple shell script
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/create-or-update?view=rest-compute-2025-04-01&tabs=HTTP
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vmName, runCommandName, armcompute.VirtualMachineRunCommand{
		Location: ptr.To(location),
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			Source: &armcompute.VirtualMachineRunCommandScriptSource{
				Script: ptr.To("#!/bin/bash\necho 'Hello from Overmind integration test'\n"),
			},
			AsyncExecution:   ptr.To(false),
			RunAsUser:        ptr.To("azureuser"),
			TimeoutInSeconds: ptr.To[int32](3600),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-virtual-machine-run-command"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Virtual machine run command %s already exists (conflict), skipping creation", runCommandName)
			return nil
		}
		return fmt.Errorf("failed to begin creating virtual machine run command: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual machine run command: %w", err)
	}

	log.Printf("Virtual machine run command %s created successfully", runCommandName)
	return nil
}

// waitForRunCommandAvailable polls until the run command is available via the Get API
func waitForRunCommandAvailable(ctx context.Context, client *armcompute.VirtualMachineRunCommandsClient, resourceGroupName, vmName, runCommandName string) error {
	maxAttempts := 10
	pollInterval := 5 * time.Second

	log.Printf("Waiting for run command %s to be available via API...", runCommandName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.GetByVirtualMachine(ctx, resourceGroupName, vmName, runCommandName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Run command %s not yet available (attempt %d/%d), waiting %v...", runCommandName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking run command availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Run command %s is available with provisioning state: %s", runCommandName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("Run command provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Run command %s provisioning state: %s (attempt %d/%d), waiting...", runCommandName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Run command exists but no provisioning state - consider it available
		log.Printf("Run command %s is available", runCommandName)
		return nil
	}

	return fmt.Errorf("timeout waiting for run command %s to be available after %d attempts", runCommandName, maxAttempts)
}

// deleteVirtualMachineRunCommand deletes an Azure virtual machine run command
func deleteVirtualMachineRunCommand(ctx context.Context, client *armcompute.VirtualMachineRunCommandsClient, resourceGroupName, vmName, runCommandName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, vmName, runCommandName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual machine run command %s not found, skipping deletion", runCommandName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting virtual machine run command: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual machine run command: %w", err)
	}

	log.Printf("Virtual machine run command %s deleted successfully", runCommandName)
	return nil
}

// deleteVirtualMachineForRunCommand deletes an Azure virtual machine
func deleteVirtualMachineForRunCommand(ctx context.Context, client *armcompute.VirtualMachinesClient, resourceGroupName, vmName string) error {
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

// deleteNetworkInterfaceForRunCommand deletes an Azure network interface with retry logic
func deleteNetworkInterfaceForRunCommand(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName string) error {
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

// deleteVirtualNetworkForRunCommand deletes an Azure virtual network
func deleteVirtualNetworkForRunCommand(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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
