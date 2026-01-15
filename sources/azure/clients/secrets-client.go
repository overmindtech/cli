package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

//go:generate mockgen -destination=../shared/mocks/mock_secrets_client.go -package=mocks -source=secrets-client.go

// SecretsPager is a type alias for the generic Pager interface with secret response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type SecretsPager = Pager[armkeyvault.SecretsClientListResponse]

// SecretsClient is an interface for interacting with Azure secrets
type SecretsClient interface {
	NewListPager(resourceGroupName string, vaultName string, options *armkeyvault.SecretsClientListOptions) SecretsPager
	Get(ctx context.Context, resourceGroupName string, vaultName string, secretName string, options *armkeyvault.SecretsClientGetOptions) (armkeyvault.SecretsClientGetResponse, error)
}

type secretsClient struct {
	client *armkeyvault.SecretsClient
}

func (c *secretsClient) NewListPager(resourceGroupName string, vaultName string, options *armkeyvault.SecretsClientListOptions) SecretsPager {
	return c.client.NewListPager(resourceGroupName, vaultName, options)
}

func (c *secretsClient) Get(ctx context.Context, resourceGroupName string, vaultName string, secretName string, options *armkeyvault.SecretsClientGetOptions) (armkeyvault.SecretsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, vaultName, secretName, options)
}

// NewSecretsClient creates a new SecretsClient from the Azure SDK client
func NewSecretsClient(client *armkeyvault.SecretsClient) SecretsClient {
	return &secretsClient{client: client}
}
