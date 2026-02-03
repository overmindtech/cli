package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_tables_client.go -package=mocks -source=tables-client.go

// TablesPager is a type alias for the generic Pager interface with table response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type TablesPager = Pager[armstorage.TableClientListResponse]

// TablesClient is an interface for interacting with Azure tables
type TablesClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, tableName string) (armstorage.TableClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) TablesPager
}

type tablesClient struct {
	client *armstorage.TableClient
}

func (a *tablesClient) Get(ctx context.Context, resourceGroupName string, accountName string, tableName string) (armstorage.TableClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, accountName, tableName, nil)
}

func (a *tablesClient) List(ctx context.Context, resourceGroupName string, accountName string) TablesPager {
	return a.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewTablesClient creates a new TablesClient from the Azure SDK client
func NewTablesClient(client *armstorage.TableClient) TablesClient {
	return &tablesClient{client: client}
}
