package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_machine_scale_sets_client.go -package=mocks -source=virtual-machine-scale-sets-client.go

// VirtualMachineScaleSetsPager is a type alias for the generic Pager interface with virtual machine scale set response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type VirtualMachineScaleSetsPager = Pager[armcompute.VirtualMachineScaleSetsClientListResponse]

// VirtualMachineScaleSetsClient is an interface for interacting with Azure virtual machine scale sets
type VirtualMachineScaleSetsClient interface {
	NewListPager(resourceGroupName string, options *armcompute.VirtualMachineScaleSetsClientListOptions) VirtualMachineScaleSetsPager
	Get(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, options *armcompute.VirtualMachineScaleSetsClientGetOptions) (armcompute.VirtualMachineScaleSetsClientGetResponse, error)
}

type virtualMachineScaleSetsClient struct {
	client *armcompute.VirtualMachineScaleSetsClient
}

func (a *virtualMachineScaleSetsClient) NewListPager(resourceGroupName string, options *armcompute.VirtualMachineScaleSetsClientListOptions) VirtualMachineScaleSetsPager {
	return a.client.NewListPager(resourceGroupName, options)
}

func (a *virtualMachineScaleSetsClient) Get(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, options *armcompute.VirtualMachineScaleSetsClientGetOptions) (armcompute.VirtualMachineScaleSetsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, virtualMachineScaleSetName, options)
}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient from the Azure SDK client
func NewVirtualMachineScaleSetsClient(client *armcompute.VirtualMachineScaleSetsClient) VirtualMachineScaleSetsClient {
	return &virtualMachineScaleSetsClient{client: client}
}
