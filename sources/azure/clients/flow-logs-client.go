package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

//go:generate mockgen -destination=../shared/mocks/mock_flow_logs_client.go -package=mocks -source=flow-logs-client.go

// FlowLogsPager is a type alias for the generic Pager interface with flow logs list response type.
type FlowLogsPager = Pager[armnetwork.FlowLogsClientListResponse]

// FlowLogsClient is an interface for interacting with Azure flow logs (child of network watcher).
type FlowLogsClient interface {
	Get(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientGetOptions) (armnetwork.FlowLogsClientGetResponse, error)
	NewListPager(resourceGroupName string, networkWatcherName string, options *armnetwork.FlowLogsClientListOptions) FlowLogsPager
}

type flowLogsClient struct {
	client *armnetwork.FlowLogsClient
}

func (a *flowLogsClient) Get(ctx context.Context, resourceGroupName string, networkWatcherName string, flowLogName string, options *armnetwork.FlowLogsClientGetOptions) (armnetwork.FlowLogsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, networkWatcherName, flowLogName, options)
}

func (a *flowLogsClient) NewListPager(resourceGroupName string, networkWatcherName string, options *armnetwork.FlowLogsClientListOptions) FlowLogsPager {
	return a.client.NewListPager(resourceGroupName, networkWatcherName, options)
}

// NewFlowLogsClient creates a new FlowLogsClient from the Azure SDK client.
func NewFlowLogsClient(client *armnetwork.FlowLogsClient) FlowLogsClient {
	return &flowLogsClient{client: client}
}
