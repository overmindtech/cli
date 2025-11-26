package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

//go:generate mockgen -destination=../shared/mocks/mock_virtual_machines_client.go -package=mocks -source=virtual-machines-client.go

// VirtualMachinesPager is an interface for paging through virtual machine results
type VirtualMachinesPager interface {
	More() bool
	NextPage(ctx context.Context) (armcompute.VirtualMachinesClientListResponse, error)
}

// VirtualMachinesClient is an interface for interacting with Azure virtual machines
type VirtualMachinesClient interface {
	Get(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (armcompute.VirtualMachinesClientGetResponse, error)
	NewListPager(resourceGroupName string, options *armcompute.VirtualMachinesClientListOptions) VirtualMachinesPager
}

// virtualMachinesClientAdapter adapts the concrete Azure SDK client to our interface
type virtualMachinesClientAdapter struct {
	client *armcompute.VirtualMachinesClient
}

func (a *virtualMachinesClientAdapter) Get(ctx context.Context, resourceGroupName string, vmName string, options *armcompute.VirtualMachinesClientGetOptions) (armcompute.VirtualMachinesClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, vmName, options)
}

func (a *virtualMachinesClientAdapter) NewListPager(resourceGroupName string, options *armcompute.VirtualMachinesClientListOptions) VirtualMachinesPager {
	return a.client.NewListPager(resourceGroupName, options)
}

// NewVirtualMachinesClient creates a new VirtualMachinesClient from the Azure SDK client
func NewVirtualMachinesClient(client *armcompute.VirtualMachinesClient) VirtualMachinesClient {
	return &virtualMachinesClientAdapter{client: client}
}
