package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
)

//go:generate mockgen -destination=../shared/mocks/mock_queues_client.go -package=mocks -source=queues-client.go

// QueuesPager is a type alias for the generic Pager interface with queue response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type QueuesPager = Pager[armstorage.QueueClientListResponse]

// QueuesClient is an interface for interacting with Azure queues
type QueuesClient interface {
	Get(ctx context.Context, resourceGroupName string, accountName string, queueName string) (armstorage.QueueClientGetResponse, error)
	List(ctx context.Context, resourceGroupName string, accountName string) QueuesPager
}

type queuesClient struct {
	client *armstorage.QueueClient
}

func (a *queuesClient) Get(ctx context.Context, resourceGroupName string, accountName string, queueName string) (armstorage.QueueClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, accountName, queueName, nil)
}

func (a *queuesClient) List(ctx context.Context, resourceGroupName string, accountName string) QueuesPager {
	return a.client.NewListPager(resourceGroupName, accountName, nil)
}

// NewQueuesClient creates a new QueuesClient from the Azure SDK client
func NewQueuesClient(client *armstorage.QueueClient) QueuesClient {
	return &queuesClient{client: client}
}
