package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
)

//go:generate mockgen -destination=../shared/mocks/mock_operational_insights_workspace_client.go -package=mocks -source=operational-insights-workspace-client.go

// OperationalInsightsWorkspacePager is a type alias for the generic Pager interface with workspace response type.
// This uses the generic Pager[T] interface to avoid code duplication.
type OperationalInsightsWorkspacePager = Pager[armoperationalinsights.WorkspacesClientListByResourceGroupResponse]

// OperationalInsightsWorkspaceClient is an interface for interacting with Azure Log Analytics Workspaces
type OperationalInsightsWorkspaceClient interface {
	NewListByResourceGroupPager(resourceGroupName string, options *armoperationalinsights.WorkspacesClientListByResourceGroupOptions) OperationalInsightsWorkspacePager
	Get(ctx context.Context, resourceGroupName string, workspaceName string, options *armoperationalinsights.WorkspacesClientGetOptions) (armoperationalinsights.WorkspacesClientGetResponse, error)
}

type operationalInsightsWorkspaceClient struct {
	client *armoperationalinsights.WorkspacesClient
}

func (o *operationalInsightsWorkspaceClient) NewListByResourceGroupPager(resourceGroupName string, options *armoperationalinsights.WorkspacesClientListByResourceGroupOptions) OperationalInsightsWorkspacePager {
	return o.client.NewListByResourceGroupPager(resourceGroupName, options)
}

func (o *operationalInsightsWorkspaceClient) Get(ctx context.Context, resourceGroupName string, workspaceName string, options *armoperationalinsights.WorkspacesClientGetOptions) (armoperationalinsights.WorkspacesClientGetResponse, error) {
	return o.client.Get(ctx, resourceGroupName, workspaceName, options)
}

// NewOperationalInsightsWorkspaceClient creates a new OperationalInsightsWorkspaceClient from the Azure SDK client
func NewOperationalInsightsWorkspaceClient(client *armoperationalinsights.WorkspacesClient) OperationalInsightsWorkspaceClient {
	return &operationalInsightsWorkspaceClient{client: client}
}
