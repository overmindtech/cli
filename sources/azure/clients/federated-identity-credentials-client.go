package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

//go:generate mockgen -destination=../shared/mocks/mock_federated_identity_credentials_client.go -package=mocks -source=federated-identity-credentials-client.go

// FederatedIdentityCredentialsPager is a pager for listing federated identity credentials
type FederatedIdentityCredentialsPager = Pager[armmsi.FederatedIdentityCredentialsClientListResponse]

// FederatedIdentityCredentialsClient is the client interface for interacting with federated identity credentials
type FederatedIdentityCredentialsClient interface {
	NewListPager(resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) FederatedIdentityCredentialsPager
	Get(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientGetOptions) (armmsi.FederatedIdentityCredentialsClientGetResponse, error)
}

type federatedIdentityCredentialsClient struct {
	client *armmsi.FederatedIdentityCredentialsClient
}

func (f *federatedIdentityCredentialsClient) NewListPager(resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) FederatedIdentityCredentialsPager {
	return f.client.NewListPager(resourceGroupName, resourceName, options)
}

func (f *federatedIdentityCredentialsClient) Get(ctx context.Context, resourceGroupName string, resourceName string, federatedIdentityCredentialResourceName string, options *armmsi.FederatedIdentityCredentialsClientGetOptions) (armmsi.FederatedIdentityCredentialsClientGetResponse, error) {
	return f.client.Get(ctx, resourceGroupName, resourceName, federatedIdentityCredentialResourceName, options)
}

// NewFederatedIdentityCredentialsClient creates a new FederatedIdentityCredentialsClient
func NewFederatedIdentityCredentialsClient(client *armmsi.FederatedIdentityCredentialsClient) FederatedIdentityCredentialsClient {
	return &federatedIdentityCredentialsClient{client: client}
}
