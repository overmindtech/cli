package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
)

//go:generate mockgen -destination=../shared/mocks/mock_managed_hsms_client.go -package=mocks -source=managed-hsms-client.go

// ManagedHSMsPager is a type alias for the generic Pager interface with managed HSM response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type ManagedHSMsPager = Pager[armkeyvault.ManagedHsmsClientListByResourceGroupResponse]

// ManagedHSMsClient is an interface for interacting with Azure managed HSMs
type ManagedHSMsClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.ManagedHsmsClientListByResourceGroupOptions) ManagedHSMsPager
	Get(ctx context.Context, resourceGroupName string, name string, options *armkeyvault.ManagedHsmsClientGetOptions) (armkeyvault.ManagedHsmsClientGetResponse, error)
}

type managedHSMsClient struct {
	client *armkeyvault.ManagedHsmsClient
}

func (c *managedHSMsClient) Get(ctx context.Context, resourceGroupName string, name string, options *armkeyvault.ManagedHsmsClientGetOptions) (armkeyvault.ManagedHsmsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, name, options)
}

func (c *managedHSMsClient) NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.ManagedHsmsClientListByResourceGroupOptions) ManagedHSMsPager {
	return c.client.NewListByResourceGroupPager(resourceGroupName, options)
}

// NewManagedHSMsClient creates a new ManagedHSMsClient from the Azure SDK client
func NewManagedHSMsClient(client *armkeyvault.ManagedHsmsClient) ManagedHSMsClient {
	return &managedHSMsClient{client: client}
}
