package manual

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeVirtualMachineLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeVirtualMachine)

type computeVirtualMachineWrapper struct {
	client clients.VirtualMachinesClient

	*azureshared.ResourceGroupBase
}

// NewComputeVirtualMachine creates a new computeVirtualMachineWrapper instance
func NewComputeVirtualMachine(client clients.VirtualMachinesClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &computeVirtualMachineWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeVirtualMachine,
		),
	}
}

// IAMPermissions returns the IAM permissions required for this adapter
// Reference: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute
func (c computeVirtualMachineWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/virtualMachines/read",
	}
}

// PotentialLinks returns the potential links for the virtual machine wrapper
func (c computeVirtualMachineWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeDisk,
		azureshared.ComputeDiskEncryptionSet,
		azureshared.NetworkNetworkInterface,
		azureshared.NetworkPublicIPAddress,
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.ComputeAvailabilitySet,
		azureshared.ComputeProximityPlacementGroup,
		azureshared.ComputeDedicatedHostGroup,
		azureshared.ComputeCapacityReservationGroup,
		azureshared.ComputeVirtualMachineScaleSet,
		azureshared.ComputeImage,
		azureshared.ComputeSharedGalleryImage,
		azureshared.ComputeSharedGalleryApplicationVersion,
		azureshared.ComputeVirtualMachineExtension,
		azureshared.ComputeVirtualMachineRunCommand,
		azureshared.ManagedIdentityUserAssignedIdentity,
		azureshared.KeyVaultVault,
		stdlib.NetworkHTTP,
		stdlib.NetworkDNS,
	)
}

// TerraformMappings returns the Terraform mappings for the virtual machine wrapper
func (c computeVirtualMachineWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_machine
			TerraformQueryMap: "azurerm_virtual_machine.name",
		},
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine
			TerraformQueryMap: "azurerm_linux_virtual_machine.name",
		},
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/windows_virtual_machine
			TerraformQueryMap: "azurerm_windows_virtual_machine.name",
		},
	}
}

// GetLookups returns the lookups for the virtual machine wrapper
// This defines how the source can be queried for specific item
// In this case, it will be: azure-compute-virtual-machine-name
func (c computeVirtualMachineWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeVirtualMachineLookupByName,
	}
}

// Get retrieves a virtual machine by its name
// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
func (c computeVirtualMachineWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	vmName := queryParts[0]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	resp, err := c.client.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.azureVirtualMachineToSDPItem(&resp.VirtualMachine, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists virtual machines in the resource group and converts them to sdp.Items.
// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/list
func (c computeVirtualMachineWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	pager := c.client.NewListPager(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}

		for _, vm := range page.Value {
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureVirtualMachineToSDPItem(vm, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (c computeVirtualMachineWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = c.ResourceGroup()
	}
	pager := c.client.NewListPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, vm := range page.Value {
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureVirtualMachineToSDPItem(vm, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeVirtualMachineWrapper) azureVirtualMachineToSDPItem(vm *armcompute.VirtualMachine, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(vm, "tags")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeVirtualMachine.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(vm.Tags),
	}

	// TODO: This adapter is demon purposes only.
	// The linked items must be reviewed before using in production.

	// Link to OS disk
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disks/get
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil {
		if vm.Properties.StorageProfile.OSDisk.ManagedDisk != nil && vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID != nil {
			diskName := azureshared.ExtractResourceName(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID)
			if diskName != "" {
				scope := c.DefaultScope()
				// Check if disk is in a different resource group
				if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // If disk changes → VM affected (In: true)
						Out: true, // If VM is deleted → disk may be deleted depending on delete option (Out: true)
					},
				})
			}
			// Link to disk encryption set for OS disk
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
			if vm.Properties.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet != nil && vm.Properties.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID != nil {
				diskEncryptionSetName := azureshared.ExtractResourceName(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID)
				if diskEncryptionSetName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeDiskEncryptionSet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskEncryptionSetName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If encryption set changes → disk encryption affected (In: true)
							Out: false, // If VM is deleted → encryption set remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to data disks
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.DataDisks != nil {
		for _, dataDisk := range vm.Properties.StorageProfile.DataDisks {
			if dataDisk.ManagedDisk != nil && dataDisk.ManagedDisk.ID != nil {
				diskName := azureshared.ExtractResourceName(*dataDisk.ManagedDisk.ID)
				if diskName != "" {
					scope := c.DefaultScope()
					// Check if disk is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.ManagedDisk.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // If disk changes → VM affected (In: true)
							Out: true, // If VM is deleted → disk may be deleted depending on delete option (Out: true)
						},
					})
				}
				// Link to disk encryption set for data disk
				// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
				if dataDisk.ManagedDisk.DiskEncryptionSet != nil && dataDisk.ManagedDisk.DiskEncryptionSet.ID != nil {
					diskEncryptionSetName := azureshared.ExtractResourceName(*dataDisk.ManagedDisk.DiskEncryptionSet.ID)
					if diskEncryptionSetName != "" {
						scope := c.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.ManagedDisk.DiskEncryptionSet.ID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeDiskEncryptionSet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  diskEncryptionSetName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If encryption set changes → disk encryption affected (In: true)
								Out: false, // If VM is deleted → encryption set remains (Out: false)
							},
						})
					}
				}
			}
		}
	}

	// Link to network interfaces
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/get
	if vm.Properties != nil && vm.Properties.NetworkProfile != nil && vm.Properties.NetworkProfile.NetworkInterfaces != nil {
		for _, nic := range vm.Properties.NetworkProfile.NetworkInterfaces {
			if nic.ID != nil {
				nicName := azureshared.ExtractResourceName(*nic.ID)
				if nicName != "" {
					scope := c.DefaultScope()
					// Check if NIC is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*nic.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If NIC changes → VM network connectivity affected (In: true)
							Out: false, // If VM is deleted → NIC remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to availability set
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/availability-sets/get
	if vm.Properties != nil && vm.Properties.AvailabilitySet != nil && vm.Properties.AvailabilitySet.ID != nil {
		availabilitySetName := azureshared.ExtractResourceName(*vm.Properties.AvailabilitySet.ID)
		if availabilitySetName != "" {
			scope := c.DefaultScope()
			// Check if availability set is in a different resource group
			if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.AvailabilitySet.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeAvailabilitySet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  availabilitySetName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If availability set changes → VM placement affected (In: true)
					Out: false, // If VM is deleted → availability set remains (Out: false)
				},
			})
		}
	}

	// Link to proximity placement group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/get
	if vm.Properties != nil && vm.Properties.ProximityPlacementGroup != nil && vm.Properties.ProximityPlacementGroup.ID != nil {
		proximityPlacementGroupName := azureshared.ExtractResourceName(*vm.Properties.ProximityPlacementGroup.ID)
		if proximityPlacementGroupName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.ProximityPlacementGroup.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeProximityPlacementGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  proximityPlacementGroupName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If proximity placement group changes → VM placement affected (In: true)
					Out: false, // If VM is deleted → proximity placement group remains (Out: false)
				},
			})
		}
	}

	// Link to dedicated host group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups/get
	if vm.Properties != nil && vm.Properties.HostGroup != nil && vm.Properties.HostGroup.ID != nil {
		hostGroupName := azureshared.ExtractResourceName(*vm.Properties.HostGroup.ID)
		if hostGroupName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.HostGroup.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDedicatedHostGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  hostGroupName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If host group changes → VM host placement affected (In: true)
					Out: false, // If VM is deleted → host group remains (Out: false)
				},
			})
		}
	}

	// Link to capacity reservation group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservation-groups/get
	if vm.Properties != nil && vm.Properties.CapacityReservation != nil && vm.Properties.CapacityReservation.CapacityReservationGroup != nil && vm.Properties.CapacityReservation.CapacityReservationGroup.ID != nil {
		capacityReservationGroupName := azureshared.ExtractResourceName(*vm.Properties.CapacityReservation.CapacityReservationGroup.ID)
		if capacityReservationGroupName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.CapacityReservation.CapacityReservationGroup.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeCapacityReservationGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  capacityReservationGroupName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If capacity reservation group changes → VM capacity reservation affected (In: true)
					Out: false, // If VM is deleted → capacity reservation group remains (Out: false)
				},
			})
		}
	}

	// Link to virtual machine scale set
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get
	if vm.Properties != nil && vm.Properties.VirtualMachineScaleSet != nil && vm.Properties.VirtualMachineScaleSet.ID != nil {
		vmssName := azureshared.ExtractResourceName(*vm.Properties.VirtualMachineScaleSet.ID)
		if vmssName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.VirtualMachineScaleSet.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeVirtualMachineScaleSet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vmssName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If VMSS changes → VM configuration affected (In: true)
					Out: false, // If VM is deleted → VMSS remains (Out: false)
				},
			})
		}
	}

	// Link to managed by resource (typically VMSS)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get
	if vm.ManagedBy != nil && *vm.ManagedBy != "" {
		// Check if managedBy is a VMSS
		if strings.Contains(*vm.ManagedBy, "/virtualMachineScaleSets/") {
			vmssName := azureshared.ExtractPathParamsFromResourceID(*vm.ManagedBy, []string{"virtualMachineScaleSets"})
			if len(vmssName) > 0 && vmssName[0] != "" {
				scope := c.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.ManagedBy); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeVirtualMachineScaleSet.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vmssName[0],
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If VMSS changes → VM configuration affected (In: true)
						Out: false, // If VM is deleted → VMSS remains (Out: false)
					},
				})
			}
		}
	}

	// Link to image reference (custom image or gallery image)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/images/get
	// or https://learn.microsoft.com/en-us/rest/api/compute/gallery-image-versions/get
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.ImageReference != nil {
		if vm.Properties.StorageProfile.ImageReference.ID != nil && *vm.Properties.StorageProfile.ImageReference.ID != "" {
			imageID := *vm.Properties.StorageProfile.ImageReference.ID
			// Check if it's a gallery image or custom image
			if strings.Contains(imageID, "/galleries/") && strings.Contains(imageID, "/images/") && strings.Contains(imageID, "/versions/") {
				// Shared Gallery Image Version
				params := azureshared.ExtractPathParamsFromResourceID(imageID, []string{"galleries", "images", "versions"})
				if len(params) == 3 {
					galleryName := params[0]
					imageName := params[1]
					versionName := params[2]
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, versionName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If image version changes → VM image affected (In: true)
							Out: false, // If VM is deleted → image version remains (Out: false)
						},
					})
				}
			} else if strings.Contains(imageID, "/images/") {
				// Custom Image
				imageName := azureshared.ExtractResourceName(imageID)
				if imageName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  imageName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If image changes → VM image affected (In: true)
							Out: false, // If VM is deleted → image remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to user assigned managed identities
	// Reference: https://learn.microsoft.com/en-us/rest/api/msi/user-assigned-identities/get
	if vm.Identity != nil && vm.Identity.UserAssignedIdentities != nil {
		for identityID := range vm.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityID)
			if identityName != "" {
				scope := c.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If identity changes → VM identity access affected (In: true)
						Out: false, // If VM is deleted → identity remains (Out: false)
					},
				})
			}
		}
	}

	// Link to Key Vault from OS profile secrets
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	if vm.Properties != nil && vm.Properties.OSProfile != nil && vm.Properties.OSProfile.Secrets != nil {
		for _, secret := range vm.Properties.OSProfile.Secrets {
			if secret.SourceVault != nil && secret.SourceVault.ID != nil {
				vaultName := azureshared.ExtractResourceName(*secret.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*secret.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault changes → VM secrets access affected (In: true)
							Out: false, // If VM is deleted → Key Vault remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to Key Vault from disk encryption settings
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil {
		if vm.Properties.StorageProfile.OSDisk.EncryptionSettings != nil {
			// Link to Key Vault from DiskEncryptionKey.SourceVault.ID
			// DiskEncryptionKey is required for Azure Disk Encryption, while KeyEncryptionKey is optional
			if vm.Properties.StorageProfile.OSDisk.EncryptionSettings.DiskEncryptionKey != nil && vm.Properties.StorageProfile.OSDisk.EncryptionSettings.DiskEncryptionKey.SourceVault != nil && vm.Properties.StorageProfile.OSDisk.EncryptionSettings.DiskEncryptionKey.SourceVault.ID != nil {
				vaultName := azureshared.ExtractResourceName(*vm.Properties.StorageProfile.OSDisk.EncryptionSettings.DiskEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.StorageProfile.OSDisk.EncryptionSettings.DiskEncryptionKey.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault changes → disk encryption affected (In: true)
							Out: false, // If VM is deleted → Key Vault remains (Out: false)
						},
					})
				}
			}
			// Link to Key Vault from KeyEncryptionKey.SourceVault.ID
			if vm.Properties.StorageProfile.OSDisk.EncryptionSettings.KeyEncryptionKey != nil && vm.Properties.StorageProfile.OSDisk.EncryptionSettings.KeyEncryptionKey.SourceVault != nil && vm.Properties.StorageProfile.OSDisk.EncryptionSettings.KeyEncryptionKey.SourceVault.ID != nil {
				vaultName := azureshared.ExtractResourceName(*vm.Properties.StorageProfile.OSDisk.EncryptionSettings.KeyEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*vm.Properties.StorageProfile.OSDisk.EncryptionSettings.KeyEncryptionKey.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault changes → disk encryption affected (In: true)
							Out: false, // If VM is deleted → Key Vault remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to gallery application versions
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/gallery-application-versions/get
	if vm.Properties != nil && vm.Properties.ApplicationProfile != nil && vm.Properties.ApplicationProfile.GalleryApplications != nil {
		for _, galleryApp := range vm.Properties.ApplicationProfile.GalleryApplications {
			if galleryApp.PackageReferenceID != nil && *galleryApp.PackageReferenceID != "" {
				packageRefID := *galleryApp.PackageReferenceID
				// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/applications/{appName}/versions/{versionName}
				params := azureshared.ExtractPathParamsFromResourceID(packageRefID, []string{"galleries", "applications", "versions"})
				if len(params) == 3 {
					galleryName := params[0]
					appName := params[1]
					versionName := params[2]
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(packageRefID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryApplicationVersion.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, appName, versionName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If application version changes → VM application affected (In: true)
							Out: false, // If VM is deleted → application version remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to boot diagnostics storage URI (standard library HTTP and DNS)
	// Reference: Boot diagnostics storage is accessed via HTTP/HTTPS
	if vm.Properties != nil && vm.Properties.DiagnosticsProfile != nil && vm.Properties.DiagnosticsProfile.BootDiagnostics != nil && vm.Properties.DiagnosticsProfile.BootDiagnostics.StorageURI != nil && *vm.Properties.DiagnosticsProfile.BootDiagnostics.StorageURI != "" {
		storageURI := *vm.Properties.DiagnosticsProfile.BootDiagnostics.StorageURI
		// Extract the HTTP/HTTPS URL for standard library
		if strings.HasPrefix(storageURI, "http://") || strings.HasPrefix(storageURI, "https://") {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkHTTP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  storageURI,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If storage URI changes → boot diagnostics affected (In: true)
					Out: false, // If VM is deleted → storage URI remains (Out: false)
				},
			})
			// Extract DNS name from URL and create DNS link
			// Reference: Any attribute containing a DNS name must create a LinkedItemQuery for dns type
			dnsName := azureshared.ExtractDNSFromURL(storageURI)
			if dnsName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  dnsName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If DNS changes → boot diagnostics affected (In: true)
						Out: false, // If VM is deleted → DNS remains (Out: false)
					},
				})
			}
		}
	}

	// Link to extensions
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-extensions/list
	if vm.Resources != nil {
		for _, extension := range vm.Resources {
			if extension.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeVirtualMachineExtension.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(*vm.Name, *extension.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false, // If Extensions are deleted → VM remains functional (In: false)
						Out: true,  // If VM is deleted → Extensions become invalid/unusable (Out: true)
					},
				})
			}
		}
	}

	// Link to run commands
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/list-by-virtual-machine?view=rest-compute-2025-04-01&tabs=HTTP
	// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}/runCommands?api-version=2025-04-01
	if vm.Name != nil {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.ComputeVirtualMachineRunCommand.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *vm.Name,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  false, // If Run Commands are deleted → VM remains functional (In: false)
				Out: true,  // If VM is deleted → Run Commands become invalid/unusable (Out: true)
			},
		})
	}

	// Map provisioning state to health status
	// Reference: https://learn.microsoft.com/en-us/azure/virtual-machines/states-billing
	if vm.Properties != nil && vm.Properties.ProvisioningState != nil {
		switch *vm.Properties.ProvisioningState {
		case "Succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "Creating", "Updating", "Migrating":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Failed", "Deleting":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	return sdpItem, nil
}
