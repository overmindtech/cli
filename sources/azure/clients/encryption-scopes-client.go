package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_encryption_scopes_client.go -package=mocks -source=encryption-scopes-client.go

// EncryptionScopesPager is a type alias for the generic Pager interface with encryption scope list response type.
type EncryptionScopesPager = Pager[armstorage.EncryptionScopesClientListResponse]

// EncryptionScopesClient is an interface for interacting with Azure storage encryption scopes
type EncryptionScopesClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, encryptionScopeName string) (armstorage.EncryptionScopesClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) EncryptionScopesPager
}

type encryptionScopesClient struct {
	client *armstorage.EncryptionScopesClient
}

func (c *encryptionScopesClient) Get(ctx context.Context, resourceGroupName string, accountName string, encryptionScopeName string) (armstorage.EncryptionScopesClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, accountName, encryptionScopeName, nil)
}

func (c *encryptionScopesClient) List(ctx context.Context, resourceGroupName string, accountName string) EncryptionScopesPager {
	return c.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewEncryptionScopesClient creates a new EncryptionScopesClient from the Azure SDK client
func NewEncryptionScopesClient(client *armstorage.EncryptionScopesClient) EncryptionScopesClient {
	return &encryptionScopesClient{client: client}
}
