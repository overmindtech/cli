package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_machine_extensions_client.go -package=mocks -source=virtual-machine-extensions-client.go

// VirtualMachineExtensionsClient is an interface for interacting with Azure virtual machine extensions
type VirtualMachineExtensionsClient interface {
	List(ctx context.Context, resourceGroupName string, virtualMachineName string, options *armcompute.VirtualMachineExtensionsClientListOptions) (armcompute.VirtualMachineExtensionsClientListResponse, error)
	Get(ctx context.Context, resourceGroupName string, virtualMachineName string, vmExtensionName string, options *armcompute.VirtualMachineExtensionsClientGetOptions) (armcompute.VirtualMachineExtensionsClientGetResponse, error)
}

// virtualMachineExtensionsClientAdapter adapts the concrete Azure SDK client to our interface
type virtualMachineExtensionsClientAdapter struct {
	client *armcompute.VirtualMachineExtensionsClient
}

func (a *virtualMachineExtensionsClientAdapter) List(ctx context.Context, resourceGroupName string, virtualMachineName string, options *armcompute.VirtualMachineExtensionsClientListOptions) (armcompute.VirtualMachineExtensionsClientListResponse, error) {
	return a.client.List(ctx, resourceGroupName, virtualMachineName, options)
}

func (a *virtualMachineExtensionsClientAdapter) Get(ctx context.Context, resourceGroupName string, virtualMachineName string, vmExtensionName string, options *armcompute.VirtualMachineExtensionsClientGetOptions) (armcompute.VirtualMachineExtensionsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, virtualMachineName, vmExtensionName, options)
}

// NewVirtualMachineExtensionsClient creates a new VirtualMachineExtensionsClient from the Azure SDK client
func NewVirtualMachineExtensionsClient(client *armcompute.VirtualMachineExtensionsClient) VirtualMachineExtensionsClient {
	return &virtualMachineExtensionsClientAdapter{client: client}
}
