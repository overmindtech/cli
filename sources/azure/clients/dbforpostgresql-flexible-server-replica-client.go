package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
)

//go:generate mockgen -destination=../shared/mocks/mock_dbforpostgresql_flexible_server_replica_client.go -package=mocks -source=dbforpostgresql-flexible-server-replica-client.go

type DBforPostgreSQLFlexibleServerReplicaPager = Pager[armpostgresqlflexibleservers.ReplicasClientListByServerResponse]

type DBforPostgreSQLFlexibleServerReplicaClient interface {
	ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerReplicaPager
	Get(ctx context.Context, resourceGroupName string, replicaName string) (armpostgresqlflexibleservers.ServersClientGetResponse, error)
}

type dbforPostgreSQLFlexibleServerReplicaClient struct {
	replicasClient *armpostgresqlflexibleservers.ReplicasClient
	serversClient  *armpostgresqlflexibleservers.ServersClient
}

func (a *dbforPostgreSQLFlexibleServerReplicaClient) ListByServer(ctx context.Context, resourceGroupName string, serverName string) DBforPostgreSQLFlexibleServerReplicaPager {
	return a.replicasClient.NewListByServerPager(resourceGroupName, serverName, nil)
}

func (a *dbforPostgreSQLFlexibleServerReplicaClient) Get(ctx context.Context, resourceGroupName string, replicaName string) (armpostgresqlflexibleservers.ServersClientGetResponse, error) {
	return a.serversClient.Get(ctx, resourceGroupName, replicaName, nil)
}

func NewDBforPostgreSQLFlexibleServerReplicaClient(replicasClient *armpostgresqlflexibleservers.ReplicasClient, serversClient *armpostgresqlflexibleservers.ServersClient) DBforPostgreSQLFlexibleServerReplicaClient {
	return &dbforPostgreSQLFlexibleServerReplicaClient{
		replicasClient: replicasClient,
		serversClient:  serversClient,
	}
}
