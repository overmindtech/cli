package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_keyvault_key_client.go -package=mocks -source=keyvault-key-client.go

// KeysPager is a type alias for the generic Pager interface with keys response type.
type KeysPager = Pager[armkeyvault.KeysClientListResponse]

// KeysClient is an interface for interacting with Azure Key Vault keys
type KeysClient interface {
	NewListPager(resourceGroupName string, vaultName string, options *armkeyvault.KeysClientListOptions) KeysPager
	Get(ctx context.Context, resourceGroupName string, vaultName string, keyName string, options *armkeyvault.KeysClientGetOptions) (armkeyvault.KeysClientGetResponse, error)
}

type keysClient struct {
	client *armkeyvault.KeysClient
}

func (c *keysClient) NewListPager(resourceGroupName string, vaultName string, options *armkeyvault.KeysClientListOptions) KeysPager {
	return c.client.NewListPager(resourceGroupName, vaultName, options)
}

func (c *keysClient) Get(ctx context.Context, resourceGroupName string, vaultName string, keyName string, options *armkeyvault.KeysClientGetOptions) (armkeyvault.KeysClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, vaultName, keyName, options)
}

// NewKeysClient creates a new KeysClient from the Azure SDK client
func NewKeysClient(client *armkeyvault.KeysClient) KeysClient {
	return &keysClient{client: client}
}
