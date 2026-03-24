package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_configurations_client.go -package=mocks -source=dbforpostgresql-configurations-client.go

// PostgreSQLConfigurationsPager is a type alias for the generic Pager interface.
type PostgreSQLConfigurationsPager = Pager[armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse]

// PostgreSQLConfigurationsClient is an interface for interacting with Azure PostgreSQL Flexible Server configurations.
type PostgreSQLConfigurationsClient interface {
	Get(ctx context.Context, resourceGroupName string, serverName string, configurationName string, options *armpostgresqlflexibleservers.ConfigurationsClientGetOptions) (armpostgresqlflexibleservers.ConfigurationsClientGetResponse, error)
	NewListByServerPager(resourceGroupName string, serverName string, options *armpostgresqlflexibleservers.ConfigurationsClientListByServerOptions) PostgreSQLConfigurationsPager
}

type postgreSQLConfigurationsClient struct {
	client *armpostgresqlflexibleservers.ConfigurationsClient
}

func (c *postgreSQLConfigurationsClient) Get(ctx context.Context, resourceGroupName string, serverName string, configurationName string, options *armpostgresqlflexibleservers.ConfigurationsClientGetOptions) (armpostgresqlflexibleservers.ConfigurationsClientGetResponse, error) {
	return c.client.Get(ctx, resourceGroupName, serverName, configurationName, options)
}

func (c *postgreSQLConfigurationsClient) NewListByServerPager(resourceGroupName string, serverName string, options *armpostgresqlflexibleservers.ConfigurationsClientListByServerOptions) PostgreSQLConfigurationsPager {
	return c.client.NewListByServerPager(resourceGroupName, serverName, options)
}

// NewPostgreSQLConfigurationsClient creates a new PostgreSQLConfigurationsClient from the Azure SDK client.
func NewPostgreSQLConfigurationsClient(client *armpostgresqlflexibleservers.ConfigurationsClient) PostgreSQLConfigurationsClient {
	return &postgreSQLConfigurationsClient{client: client}
}
