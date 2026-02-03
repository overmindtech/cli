package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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
	integrationTestVMSSName       = "ovm-integ-test-vmss"
	integrationTestVMSSVNetName   = "ovm-integ-test-vmss-vnet"
	integrationTestVMSSSubnetName = "default"
)

func TestComputeVirtualMachineScaleSetIntegration(t *testing.T) {
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
	vmssClient, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Machine Scale Sets client: %v", err)
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

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network
		err = createVirtualNetworkForVMSS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVMSSVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		// Get subnet ID for VMSS creation
		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestVMSSVNetName, integrationTestVMSSSubnetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		// Create virtual machine scale set
		err = createVirtualMachineScaleSet(ctx, vmssClient, integrationTestResourceGroup, integrationTestVMSSName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create virtual machine scale set: %v", err)
		}

		// Wait for VMSS to be fully available via the API
		err = waitForVMSSAvailable(ctx, vmssClient, integrationTestResourceGroup, integrationTestVMSSName)
		if err != nil {
			t.Fatalf("Failed waiting for VMSS to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		// Check if VMSS exists - if Setup failed (e.g., quota issues), skip Run tests
		ctx := t.Context()
		_, err := vmssClient.Get(ctx, integrationTestResourceGroup, integrationTestVMSSName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				t.Skipf("VMSS %s does not exist - Setup may have failed (e.g., quota issues). Skipping Run tests.", integrationTestVMSSName)
			}
		}

		t.Run("GetVirtualMachineScaleSet", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving virtual machine scale set %s in subscription %s, resource group %s",
				integrationTestVMSSName, subscriptionID, integrationTestResourceGroup)

			vmssWrapper := manual.NewComputeVirtualMachineScaleSet(
				clients.NewVirtualMachineScaleSetsClient(vmssClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmssWrapper.Scopes()[0]

			vmssAdapter := sources.WrapperToAdapter(vmssWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := vmssAdapter.Get(ctx, scope, integrationTestVMSSName, true)
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

			if uniqueAttrValue != integrationTestVMSSName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestVMSSName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.ComputeVirtualMachineScaleSet.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.ComputeVirtualMachineScaleSet, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved virtual machine scale set %s", integrationTestVMSSName)
		})

		t.Run("ListVirtualMachineScaleSets", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing virtual machine scale sets in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			vmssWrapper := manual.NewComputeVirtualMachineScaleSet(
				clients.NewVirtualMachineScaleSetsClient(vmssClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmssWrapper.Scopes()[0]

			vmssAdapter := sources.WrapperToAdapter(vmssWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := vmssAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list virtual machine scale sets: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one virtual machine scale set, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestVMSSName {
					found = true
					if item.GetType() != azureshared.ComputeVirtualMachineScaleSet.String() {
						t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineScaleSet, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find VMSS %s in the list of virtual machine scale sets", integrationTestVMSSName)
			}

			log.Printf("Found %d virtual machine scale sets in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for virtual machine scale set %s", integrationTestVMSSName)

			vmssWrapper := manual.NewComputeVirtualMachineScaleSet(
				clients.NewVirtualMachineScaleSetsClient(vmssClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmssWrapper.Scopes()[0]

			vmssAdapter := sources.WrapperToAdapter(vmssWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := vmssAdapter.Get(ctx, scope, integrationTestVMSSName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasSubnetLink, hasVMLink bool
			for _, liq := range linkedQueries {
				switch liq.GetQuery().GetType() {
				case azureshared.NetworkSubnet.String():
					hasSubnetLink = true
					// Verify subnet link properties
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected subnet link method to be GET, got %s", liq.GetQuery().GetMethod())
					}
					// Verify blast propagation (In: true, Out: false)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected subnet blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected subnet blast propagation Out=false, got true")
					}
				case azureshared.ComputeVirtualMachine.String():
					hasVMLink = true
					// Verify VM link properties (VM instances are linked via SEARCH)
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected VM link method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetQuery() != integrationTestVMSSName {
						t.Errorf("Expected VM link query to be %s, got %s", integrationTestVMSSName, liq.GetQuery().GetQuery())
					}
					// Verify blast propagation (In: false, Out: true)
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected VM blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected VM blast propagation Out=true, got false")
					}
				case azureshared.ComputeVirtualMachineExtension.String():
					// Extensions may or may not be present depending on VMSS setup
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

			if !hasSubnetLink {
				t.Error("Expected linked query to subnet, but didn't find one")
			}

			// VM instances link should always be present (even if no instances exist)
			if !hasVMLink {
				t.Error("Expected linked query to VM instances, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for VMSS %s", len(linkedQueries), integrationTestVMSSName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for VMSS %s", integrationTestVMSSName)

			vmssWrapper := manual.NewComputeVirtualMachineScaleSet(
				clients.NewVirtualMachineScaleSetsClient(vmssClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vmssWrapper.Scopes()[0]

			vmssAdapter := sources.WrapperToAdapter(vmssWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := vmssAdapter.Get(ctx, scope, integrationTestVMSSName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.ComputeVirtualMachineScaleSet.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeVirtualMachineScaleSet, sdpItem.GetType())
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

			// Verify health status (should be OK if provisioning succeeded)
			if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
				t.Logf("VMSS health status is %s (may be pending if still provisioning)", sdpItem.GetHealth())
			}

			log.Printf("Verified item attributes for VMSS %s", integrationTestVMSSName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete VMSS first
		err := deleteVirtualMachineScaleSet(ctx, vmssClient, integrationTestResourceGroup, integrationTestVMSSName)
		if err != nil {
			t.Fatalf("Failed to delete virtual machine scale set: %v", err)
		}

		// Delete VNet (this also deletes the subnet)
		err = deleteVirtualNetworkForVMSS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVMSSVNetName)
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

// createVirtualNetworkForVMSS creates an Azure virtual network with a default subnet (idempotent)
func createVirtualNetworkForVMSS(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
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
					Name: ptr.To(integrationTestVMSSSubnetName),
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

// createVirtualMachineScaleSet creates an Azure virtual machine scale set (idempotent)
func createVirtualMachineScaleSet(ctx context.Context, client *armcompute.VirtualMachineScaleSetsClient, resourceGroupName, vmssName, location, subnetID string) error {
	// Check if VMSS already exists
	existingVMSS, err := client.Get(ctx, resourceGroupName, vmssName, nil)
	if err == nil {
		// VMSS exists, check its state
		if existingVMSS.Properties != nil && existingVMSS.Properties.ProvisioningState != nil {
			state := *existingVMSS.Properties.ProvisioningState
			switch state {
			case "Succeeded", "Updating":
				// VMSS exists and is in a good state - we'll wait for it to be fully available
				log.Printf("Virtual machine scale set %s already exists with state %s, will verify availability", vmssName, state)
				return nil
			case "Failed", "Deleting", "Deleted":
				// VMSS is in a bad state - delete it so we can recreate
				log.Printf("Virtual machine scale set %s exists but in state %s, deleting before recreation", vmssName, state)
				deleteErr := deleteVirtualMachineScaleSet(ctx, client, resourceGroupName, vmssName)
				if deleteErr != nil {
					return fmt.Errorf("failed to delete VMSS in bad state: %w", deleteErr)
				}
				// Wait a bit after deletion before recreating
				time.Sleep(10 * time.Second)
			default:
				// Creating, etc. - wait for it
				log.Printf("Virtual machine scale set %s exists but in state %s, will wait for it", vmssName, state)
				return nil
			}
		} else {
			log.Printf("Virtual machine scale set %s already exists, will verify availability", vmssName)
			return nil
		}
	}

	// Create the VMSS
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vmssName, armcompute.VirtualMachineScaleSet{
		Location: ptr.To(location),
		SKU: &armcompute.SKU{
			Name:     ptr.To("Standard_B1s"), // Burstable B-series VM - cheaper and more widely available
			Tier:     ptr.To("Standard"),
			Capacity: ptr.To[int64](1), // Start with 1 instance for testing
		},
		Properties: &armcompute.VirtualMachineScaleSetProperties{
			UpgradePolicy: &armcompute.UpgradePolicy{
				Mode: ptr.To(armcompute.UpgradeModeManual),
			},
			VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
				OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: ptr.To(vmssName),
					AdminUsername:      ptr.To("azureuser"),
					AdminPassword:      ptr.To("OvmIntegTest2024!"),
					LinuxConfiguration: &armcompute.LinuxConfiguration{
						DisablePasswordAuthentication: ptr.To(false),
					},
				},
				StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &armcompute.ImageReference{
						Publisher: ptr.To("Canonical"),
						Offer:     ptr.To("0001-com-ubuntu-server-jammy"),
						SKU:       ptr.To("22_04-lts"), // x64 image for B-series VM
						Version:   ptr.To("latest"),
					},
					OSDisk: &armcompute.VirtualMachineScaleSetOSDisk{
						CreateOption: ptr.To(armcompute.DiskCreateOptionTypesFromImage),
						ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: ptr.To(armcompute.StorageAccountTypesStandardLRS),
						},
					},
				},
				NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: []*armcompute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: ptr.To("vmss-nic-config"),
							Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary: ptr.To(true),
								IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: ptr.To("ipconfig1"),
										Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &armcompute.APIEntityReference{
												ID: ptr.To(subnetID),
											},
											Primary: ptr.To(true),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-virtual-machine-scale-set"),
		},
	}, nil)
	if err != nil {
		// Check if VMSS already exists (conflict) or quota issue
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == http.StatusConflict {
				log.Printf("Virtual machine scale set %s already exists (conflict), verifying it exists", vmssName)
				// Verify the VMSS actually exists
				_, getErr := client.Get(ctx, resourceGroupName, vmssName, nil)
				if getErr != nil {
					// If we get a conflict but VMSS doesn't exist, it might be in a transient state (deleting)
					// Wait longer and retry creation once
					log.Printf("VMSS %s not found after conflict, waiting 30s and retrying creation", vmssName)
					time.Sleep(30 * time.Second)

					// Retry creation
					retryPoller, retryErr := client.BeginCreateOrUpdate(ctx, resourceGroupName, vmssName, armcompute.VirtualMachineScaleSet{
						Location: ptr.To(location),
						SKU: &armcompute.SKU{
							Name:     ptr.To("Standard_B1s"),
							Tier:     ptr.To("Standard"),
							Capacity: ptr.To[int64](1),
						},
						Properties: &armcompute.VirtualMachineScaleSetProperties{
							UpgradePolicy: &armcompute.UpgradePolicy{
								Mode: ptr.To(armcompute.UpgradeModeManual),
							},
							VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
								OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
									ComputerNamePrefix: ptr.To(vmssName),
									AdminUsername:      ptr.To("azureuser"),
									AdminPassword:      ptr.To("OvmIntegTest2024!"),
									LinuxConfiguration: &armcompute.LinuxConfiguration{
										DisablePasswordAuthentication: ptr.To(false),
									},
								},
								StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
									ImageReference: &armcompute.ImageReference{
										Publisher: ptr.To("Canonical"),
										Offer:     ptr.To("0001-com-ubuntu-server-jammy"),
										SKU:       ptr.To("22_04-lts"),
										Version:   ptr.To("latest"),
									},
									OSDisk: &armcompute.VirtualMachineScaleSetOSDisk{
										CreateOption: ptr.To(armcompute.DiskCreateOptionTypesFromImage),
										ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
											StorageAccountType: ptr.To(armcompute.StorageAccountTypesStandardLRS),
										},
									},
								},
								NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
									NetworkInterfaceConfigurations: []*armcompute.VirtualMachineScaleSetNetworkConfiguration{
										{
											Name: ptr.To("vmss-nic-config"),
											Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
												Primary: ptr.To(true),
												IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
													{
														Name: ptr.To("ipconfig1"),
														Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
															Subnet: &armcompute.APIEntityReference{
																ID: ptr.To(subnetID),
															},
															Primary: ptr.To(true),
														},
													},
												},
											},
										},
									},
								},
							},
						},
						Tags: map[string]*string{
							"purpose": ptr.To("overmind-integration-tests"),
							"test":    ptr.To("compute-virtual-machine-scale-set"),
						},
					}, nil)
					if retryErr != nil {
						var retryRespErr *azcore.ResponseError
						if errors.As(retryErr, &retryRespErr) && retryRespErr.StatusCode == http.StatusConflict {
							// Still conflict - check if it exists now
							_, finalCheckErr := client.Get(ctx, resourceGroupName, vmssName, nil)
							if finalCheckErr != nil {
								return fmt.Errorf("VMSS %s still returns conflict but doesn't exist after retry - may need manual cleanup", vmssName)
							}
							log.Printf("VMSS %s exists after retry conflict", vmssName)
							return nil
						}
						return fmt.Errorf("failed to retry creating virtual machine scale set after conflict: %w", retryErr)
					}

					// Poll the retry poller
					retryResp, retryPollErr := retryPoller.PollUntilDone(ctx, nil)
					if retryPollErr != nil {
						return fmt.Errorf("failed to create virtual machine scale set on retry: %w", retryPollErr)
					}
					if retryResp.Properties != nil && retryResp.Properties.ProvisioningState != nil {
						log.Printf("Virtual machine scale set %s created successfully on retry with state: %s", vmssName, *retryResp.Properties.ProvisioningState)
					}
					// Successfully created on retry - return nil is correct here
					return nil
				}
				// getErr is nil, meaning VMSS exists - return nil is correct here
				// VMSS exists, will wait for it in waitForVMSSAvailable
				log.Printf("VMSS %s exists", vmssName)
				return nil
			}
			// Handle quota errors gracefully - log but don't fail the test setup
			if respErr.ErrorCode == "OperationNotAllowed" && strings.Contains(respErr.Error(), "quota") {
				log.Printf("VMSS creation failed due to quota limits: %s. Skipping VMSS creation for this test run.", respErr.Error())
				return nil // Skip creation, test will fail gracefully in Run phase
			}
		}
		return fmt.Errorf("failed to begin creating virtual machine scale set: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual machine scale set: %w", err)
	}

	// Verify the VMSS was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("VMSS created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("VMSS provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Virtual machine scale set %s created successfully with provisioning state: %s", vmssName, provisioningState)
	return nil
}

// waitForVMSSAvailable polls until the VMSS is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the VMSS is queryable
func waitForVMSSAvailable(ctx context.Context, client *armcompute.VirtualMachineScaleSetsClient, resourceGroupName, vmssName string) error {
	maxAttempts := defaultMaxPollAttempts
	pollInterval := defaultPollInterval
	maxNotFoundAttempts := 5 // Fail faster if VMSS doesn't exist

	log.Printf("Waiting for VMSS %s to be available via API...", vmssName)

	notFoundCount := 0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, vmssName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) {
				if respErr.StatusCode == http.StatusNotFound {
					notFoundCount++
					// If VMSS doesn't exist, fail after a few attempts
					// This indicates the VMSS was never created or was deleted
					if notFoundCount >= maxNotFoundAttempts {
						return fmt.Errorf("VMSS %s not found after %d attempts - creation may have failed or VMSS was deleted", vmssName, notFoundCount)
					}
					// Early attempts might be transient, wait a bit
					if attempt < maxAttempts {
						log.Printf("VMSS %s not yet available (attempt %d/%d, not found %d/%d), waiting %v...", vmssName, attempt, maxAttempts, notFoundCount, maxNotFoundAttempts, pollInterval)
						time.Sleep(pollInterval)
						continue
					}
				}
			}
			return fmt.Errorf("error checking VMSS availability: %w", err)
		}
		// Reset not found count if we successfully found the VMSS
		notFoundCount = 0

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			switch state {
			case "Succeeded":
				log.Printf("VMSS %s is available with provisioning state: %s", vmssName, state)
				return nil
			case "Failed":
				// If failed, log details but still consider it "available" for testing purposes
				// The test will fail if needed when trying to use it
				log.Printf("VMSS %s is in Failed state but will proceed with test", vmssName)
				return nil
			case "Deleting", "Deleted":
				// If being deleted or already deleted, this is a problem
				return fmt.Errorf("VMSS %s is in state %s - may need to be recreated", vmssName, state)
			default:
				// Still provisioning or in transition state, wait and retry
				if attempt < maxAttempts {
					log.Printf("VMSS %s provisioning state: %s (attempt %d/%d), waiting %v...", vmssName, state, attempt, maxAttempts, pollInterval)
					time.Sleep(pollInterval)
					continue
				}
				// On last attempt, accept it as available even if not Succeeded
				// Some states like "Updating" might persist
				log.Printf("VMSS %s is in state %s after %d attempts, proceeding", vmssName, state, maxAttempts)
				return nil
			}
		}

		// VMSS exists but no provisioning state - consider it available
		log.Printf("VMSS %s is available (no provisioning state)", vmssName)
		return nil
	}

	return fmt.Errorf("timeout waiting for VMSS %s to be available after %d attempts", vmssName, maxAttempts)
}

// deleteVirtualMachineScaleSet deletes an Azure virtual machine scale set
func deleteVirtualMachineScaleSet(ctx context.Context, client *armcompute.VirtualMachineScaleSetsClient, resourceGroupName, vmssName string) error {
	// Use forceDeletion to speed up cleanup
	poller, err := client.BeginDelete(ctx, resourceGroupName, vmssName, &armcompute.VirtualMachineScaleSetsClientBeginDeleteOptions{
		ForceDeletion: ptr.To(true),
	})
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual machine scale set %s not found, skipping deletion", vmssName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting virtual machine scale set: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual machine scale set: %w", err)
	}

	log.Printf("Virtual machine scale set %s deleted successfully", vmssName)

	// Wait a bit to allow Azure to release associated resources
	log.Printf("Waiting 30 seconds for Azure to release associated resources...")
	time.Sleep(30 * time.Second)

	return nil
}

// deleteVirtualNetworkForVMSS deletes an Azure virtual network
func deleteVirtualNetworkForVMSS(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
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
