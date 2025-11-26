package manual

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
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
		azureshared.NetworkNetworkInterface,
		azureshared.NetworkPublicIPAddress,
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.ComputeAvailabilitySet,
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
func (c computeVirtualMachineWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	vmName := queryParts[0]

	resp, err := c.client.Get(ctx, c.ResourceGroup(), vmName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.azureVirtualMachineToSDPItem(&resp.VirtualMachine)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists virtual machines in the resource group and converts them to sdp.Items.
// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/list
func (c computeVirtualMachineWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		for _, vm := range page.Value {
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureVirtualMachineToSDPItem(vm)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (c computeVirtualMachineWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		for _, vm := range page.Value {
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureVirtualMachineToSDPItem(vm)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeVirtualMachineWrapper) azureVirtualMachineToSDPItem(vm *armcompute.VirtualMachine) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.DefaultScope(),
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
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  c.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// Link to data disks
	if vm.Properties != nil && vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.DataDisks != nil {
		for _, dataDisk := range vm.Properties.StorageProfile.DataDisks {
			if dataDisk.ManagedDisk != nil && dataDisk.ManagedDisk.ID != nil {
				diskName := azureshared.ExtractResourceName(*dataDisk.ManagedDisk.ID)
				if diskName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskName,
							Scope:  c.DefaultScope(),
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
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
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName,
							Scope:  c.DefaultScope(),
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
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
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeAvailabilitySet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  availabilitySetName,
					Scope:  c.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
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
