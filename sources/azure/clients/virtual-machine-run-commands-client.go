package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_machine_run_commands_client.go -package=mocks -source=virtual-machine-run-commands-client.go

// VirtualMachineRunCommandsPager is a type alias for the generic Pager interface with virtual machine run command response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type VirtualMachineRunCommandsPager = Pager[armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse]

type VirtualMachineRunCommandsClient interface {
	NewListByVirtualMachinePager(resourceGroupName string, virtualMachineName string, options *armcompute.VirtualMachineRunCommandsClientListByVirtualMachineOptions) VirtualMachineRunCommandsPager
	GetByVirtualMachine(ctx context.Context, resourceGroupName string, virtualMachineName string, runCommandName string, options *armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineOptions) (armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse, error)
}

type virtualMachineRunCommandsClient struct {
	client *armcompute.VirtualMachineRunCommandsClient
}

func (a *virtualMachineRunCommandsClient) NewListByVirtualMachinePager(resourceGroupName string, virtualMachineName string, options *armcompute.VirtualMachineRunCommandsClientListByVirtualMachineOptions) VirtualMachineRunCommandsPager {
	return a.client.NewListByVirtualMachinePager(resourceGroupName, virtualMachineName, options)
}

func (a *virtualMachineRunCommandsClient) GetByVirtualMachine(
	ctx context.Context,
	resourceGroupName string,
	virtualMachineName string,
	runCommandName string,
	options *armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineOptions,
) (armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse, error) {
	return a.client.GetByVirtualMachine(
		ctx,
		resourceGroupName,
		virtualMachineName,
		runCommandName,
		options,
	)
}

// NewVirtualMachineRunCommandsClient creates a new VirtualMachineRunCommandsClient from the Azure SDK client
func NewVirtualMachineRunCommandsClient(client *armcompute.VirtualMachineRunCommandsClient) VirtualMachineRunCommandsClient {
	return &virtualMachineRunCommandsClient{client: client}
}
