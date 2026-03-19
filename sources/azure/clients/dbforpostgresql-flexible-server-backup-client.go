package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_flexible_server_backup_client.go -package=mocks -source=dbforpostgresql-flexible-server-backup-client.go

type DBforPostgreSQLFlexibleServerBackupPager = Pager[armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse]

type DBforPostgreSQLFlexibleServerBackupClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerBackupPager
	Get(ctx context.Context, resourceGroupName string, serverName string, backupName string) (armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientGetResponse, error)
}

type dbforPostgreSQLFlexibleServerBackupClient struct {
	client *armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClient
}

func (a *dbforPostgreSQLFlexibleServerBackupClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerBackupPager {
	return a.client.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *dbforPostgreSQLFlexibleServerBackupClient) Get(ctx context.Context, resourceGroupName string, serverName string, backupName string) (armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, serverName, backupName, nil)
}

func NewDBforPostgreSQLFlexibleServerBackupClient(client *armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClient) DBforPostgreSQLFlexibleServerBackupClient {
	return &dbforPostgreSQLFlexibleServerBackupClient{client: client}
}
