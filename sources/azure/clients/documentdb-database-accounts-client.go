package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
)

//go:generate mockgen -destination=../shared/mocks/mock_documentdb_database_accounts_client.go -package=mocks -source=documentdb-database-accounts-client.go

// DocumentDBDatabaseAccountsPager is a type alias for the generic Pager interface with documentdb database account response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type DocumentDBDatabaseAccountsPager = Pager[armcosmos.DatabaseAccountsClientListByResourceGroupResponse]

// DocumentDBDatabaseAccountsClient is an interface for interacting with Azure documentdb database accounts
type DocumentDBDatabaseAccountsClient interface {
	ListByResourceGroup(resourceGroupName string) DocumentDBDatabaseAccountsPager
	Get(ctx context.Context, resourceGroupName string, accountName string) (armcosmos.DatabaseAccountsClientGetResponse, error)
}

type documentDBDatabaseAccountsClient struct {
	client *armcosmos.DatabaseAccountsClient
}

func (a *documentDBDatabaseAccountsClient) ListByResourceGroup(resourceGroupName string) DocumentDBDatabaseAccountsPager {
	return a.client.NewListByResourceGroupPager(resourceGroupName, nil)
}

func (a *documentDBDatabaseAccountsClient) Get(ctx context.Context, resourceGroupName string, accountName string) (armcosmos.DatabaseAccountsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, accountName, nil)
}

// NewDocumentDBDatabaseAccountsClient creates a new DocumentDBDatabaseAccountsClient from the Azure SDK client
func NewDocumentDBDatabaseAccountsClient(client *armcosmos.DatabaseAccountsClient) DocumentDBDatabaseAccountsClient {
	return &documentDBDatabaseAccountsClient{client: client}
}
