package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
)

//go:generate mockgen -destination=../shared/mocks/mock_keyvault_managed_hsm_private_endpoint_connection_client.go -package=mocks -source=keyvault-managed-hsm-private-endpoint-connection-client.go

// KeyVaultManagedHSMPrivateEndpointConnectionsPager is a type alias for the generic Pager interface with MHSM private endpoint connection list response type.
type KeyVaultManagedHSMPrivateEndpointConnectionsPager = Pager[armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse]

// KeyVaultManagedHSMPrivateEndpointConnectionsClient is an interface for interacting with Azure Key Vault Managed HSM private endpoint connections.
type KeyVaultManagedHSMPrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, hsmName string, privateEndpointConnectionName string) (armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse, error)
	ListByResource(ctx context.Context, resourceGroupName string, hsmName string) KeyVaultManagedHSMPrivateEndpointConnectionsPager
}

type keyvaultManagedHSMPrivateEndpointConnectionsClient struct {
	client *armkeyvault.MHSMPrivateEndpointConnectionsClient
}

func (c *keyvaultManagedHSMPrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, hsmName string, privateEndpointConnectionName string) (armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, hsmName, privateEndpointConnectionName, nil)
}

func (c *keyvaultManagedHSMPrivateEndpointConnectionsClient) ListByResource(ctx context.Context, resourceGroupName string, hsmName string) KeyVaultManagedHSMPrivateEndpointConnectionsPager {
	return c.client.NewListByResourcePager(resourceGroupName, hsmName, nil)
}

// NewKeyVaultManagedHSMPrivateEndpointConnectionsClient creates a new KeyVaultManagedHSMPrivateEndpointConnectionsClient from the Azure SDK client.
func NewKeyVaultManagedHSMPrivateEndpointConnectionsClient(client *armkeyvault.MHSMPrivateEndpointConnectionsClient) KeyVaultManagedHSMPrivateEndpointConnectionsClient {
	return &keyvaultManagedHSMPrivateEndpointConnectionsClient{client: client}
}
