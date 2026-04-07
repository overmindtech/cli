package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance"
)

//go:generate mockgen -destination=../shared/mocks/mock_maintenance_configuration_client.go -package=mocks -source=maintenance-configuration-client.go

// MaintenanceConfigurationPager is a type alias for the generic Pager interface with maintenance configuration response type.
type MaintenanceConfigurationPager = Pager[armmaintenance.ConfigurationsForResourceGroupClientListResponse]

// MaintenanceConfigurationClient is an interface for interacting with Azure maintenance configurations
type MaintenanceConfigurationClient interface {
	NewListPager(resourceGroupName string, options *armmaintenance.ConfigurationsForResourceGroupClientListOptions) MaintenanceConfigurationPager
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmaintenance.ConfigurationsClientGetOptions) (armmaintenance.ConfigurationsClientGetResponse, error)
}

type maintenanceConfigurationClient struct {
	configurationsClient                 *armmaintenance.ConfigurationsClient
	configurationsForResourceGroupClient *armmaintenance.ConfigurationsForResourceGroupClient
}

func (c *maintenanceConfigurationClient) NewListPager(resourceGroupName string, options *armmaintenance.ConfigurationsForResourceGroupClientListOptions) MaintenanceConfigurationPager {
	return c.configurationsForResourceGroupClient.NewListPager(resourceGroupName, options)
}

func (c *maintenanceConfigurationClient) Get(ctx context.Context, resourceGroupName string, resourceName string, options *armmaintenance.ConfigurationsClientGetOptions) (armmaintenance.ConfigurationsClientGetResponse, error) {
	return c.configurationsClient.Get(ctx, resourceGroupName, resourceName, options)
}

// NewMaintenanceConfigurationClient creates a new MaintenanceConfigurationClient from the Azure SDK clients
func NewMaintenanceConfigurationClient(configurationsClient *armmaintenance.ConfigurationsClient, configurationsForResourceGroupClient *armmaintenance.ConfigurationsForResourceGroupClient) MaintenanceConfigurationClient {
	return &maintenanceConfigurationClient{
		configurationsClient:                 configurationsClient,
		configurationsForResourceGroupClient: configurationsForResourceGroupClient,
	}
}
