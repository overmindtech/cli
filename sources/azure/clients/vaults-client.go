package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_vaults_client.go -package=mocks -source=vaults-client.go

// VaultsPager is a type alias for the generic Pager interface with vault response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type VaultsPager = Pager[armkeyvault.VaultsClientListByResourceGroupResponse]

// VaultsClient is an interface for interacting with Azure vaults
type VaultsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.VaultsClientListByResourceGroupOptions) VaultsPager
	Get(ctx context.Context, resourceGroupName string, vaultName string, options *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error)
}

type vaultsClient struct {
	client *armkeyvault.VaultsClient
}

func (c *vaultsClient) Get(ctx context.Context, resourceGroupName string, vaultName string, options *armkeyvault.VaultsClientGetOptions) (armkeyvault.VaultsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, vaultName, options)
}

func (c *vaultsClient) NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.VaultsClientListByResourceGroupOptions) VaultsPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewVaultsClient creates a new VaultsClient from the Azure SDK client
func NewVaultsClient(client *armkeyvault.VaultsClient) VaultsClient {
	return &vaultsClient{client: client}
}
