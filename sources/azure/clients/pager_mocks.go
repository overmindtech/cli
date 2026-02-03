package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

// These interfaces are defined specifically for mock generation.
// They represent the concrete Pager types used in tests.
// Since type aliases cannot be mocked directly, we define concrete interfaces
// that match the Pager[T] interface for specific types.

// VirtualMachinesPagerInterface is a concrete interface for VirtualMachinesPager to enable mock generation
//
//go:generate mockgen -destination=../shared/mocks/mock_virtual_machines_pager.go -package=mocks github.com/overmindtech/cli/sources/azure/clients VirtualMachinesPagerInterface
type VirtualMachinesPagerInterface interface {
	More() bool
	NextPage(ctx context.Context) (armcompute.VirtualMachinesClientListResponse, error)
}

// StorageAccountsPagerInterface is a concrete interface for StorageAccountsPager to enable mock generation
//
//go:generate mockgen -destination=../shared/mocks/mock_storage_accounts_pager.go -package=mocks github.com/overmindtech/cli/sources/azure/clients StorageAccountsPagerInterface
type StorageAccountsPagerInterface interface {
	More() bool
	NextPage(ctx context.Context) (armstorage.AccountsClientListByResourceGroupResponse, error)
}
