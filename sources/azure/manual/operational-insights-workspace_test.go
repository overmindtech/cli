package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestOperationalInsightsWorkspace(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		workspaceName := "test-workspace"
		workspace := createAzureWorkspace(workspaceName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, workspaceName, nil).Return(
			armoperationalinsights.WorkspacesClientGetResponse{
				Workspace: *workspace,
			}, nil)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], workspaceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.OperationalInsightsWorkspace.String() {
			t.Errorf("Expected type %s, got %s", azureshared.OperationalInsightsWorkspace, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != workspaceName {
			t.Errorf("Expected unique attribute value %s, got %s", workspaceName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		// Verify health status based on provisioning state
		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got %s", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.PrivateLinkScopedResources[0].ResourceID
					ExpectedType:   azureshared.InsightsPrivateLinkScopeScopedResource.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-pls", "test-scoped-resource"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					// Properties.Features.ClusterResourceID
					ExpectedType:   azureshared.OperationalInsightsCluster.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-cluster",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		workspaceName := "test-workspace-cross-rg"
		workspace := createAzureWorkspaceWithCrossResourceGroupLinks(workspaceName, subscriptionID)

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, workspaceName, nil).Return(
			armoperationalinsights.WorkspacesClientGetResponse{
				Workspace: *workspace,
			}, nil)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], workspaceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that links use the correct scope from different resource groups
		foundClusterLink := false
		foundPLSScopedResourceLink := false
		expectedScope := subscriptionID + ".other-rg"

		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.OperationalInsightsCluster.String() {
				foundClusterLink = true
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected Cluster scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
			if link.GetQuery().GetType() == azureshared.InsightsPrivateLinkScopeScopedResource.String() {
				foundPLSScopedResourceLink = true
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected Private Link Scope Scoped Resource scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
				expectedQuery := shared.CompositeLookupKey("test-pls-cross", "test-scoped-resource-cross")
				if link.GetQuery().GetQuery() != expectedQuery {
					t.Errorf("Expected query %s, got %s", expectedQuery, link.GetQuery().GetQuery())
				}
			}
		}

		if !foundClusterLink {
			t.Error("Expected to find Operational Insights Cluster link")
		}
		if !foundPLSScopedResourceLink {
			t.Error("Expected to find Private Link Scope Scoped Resource link")
		}
	})

	t.Run("GetWithoutLinks", func(t *testing.T) {
		workspaceName := "test-workspace-no-links"
		workspace := createAzureWorkspaceWithoutLinks(workspaceName)

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, workspaceName, nil).Return(
			armoperationalinsights.WorkspacesClientGetResponse{
				Workspace: *workspace,
			}, nil)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], workspaceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("GetWithDifferentHealthStates", func(t *testing.T) {
		healthTests := []struct {
			state          armoperationalinsights.WorkspaceEntityStatus
			expectedHealth sdp.Health
		}{
			{armoperationalinsights.WorkspaceEntityStatusSucceeded, sdp.Health_HEALTH_OK},
			{armoperationalinsights.WorkspaceEntityStatusCreating, sdp.Health_HEALTH_PENDING},
			{armoperationalinsights.WorkspaceEntityStatusUpdating, sdp.Health_HEALTH_PENDING},
			{armoperationalinsights.WorkspaceEntityStatusDeleting, sdp.Health_HEALTH_PENDING},
			{armoperationalinsights.WorkspaceEntityStatusProvisioningAccount, sdp.Health_HEALTH_PENDING},
			{armoperationalinsights.WorkspaceEntityStatusFailed, sdp.Health_HEALTH_ERROR},
			{armoperationalinsights.WorkspaceEntityStatusCanceled, sdp.Health_HEALTH_ERROR},
		}

		for _, ht := range healthTests {
			t.Run(string(ht.state), func(t *testing.T) {
				workspaceName := "test-workspace-" + string(ht.state)
				workspace := createAzureWorkspaceWithProvisioningState(workspaceName, ht.state)

				mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, workspaceName, nil).Return(
					armoperationalinsights.WorkspacesClientGetResponse{
						Workspace: *workspace,
					}, nil)

				wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], workspaceName, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != ht.expectedHealth {
					t.Errorf("Expected health %s for state %s, got %s", ht.expectedHealth, ht.state, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		workspace1 := createAzureWorkspace("test-workspace-1", subscriptionID, resourceGroup)
		workspace2 := createAzureWorkspace("test-workspace-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockPager := newMockOperationalInsightsWorkspacePager(ctrl, []*armoperationalinsights.Workspace{workspace1, workspace2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		workspace1 := createAzureWorkspace("test-workspace-1", subscriptionID, resourceGroup)
		workspace2 := createAzureWorkspace("test-workspace-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockPager := newMockOperationalInsightsWorkspacePager(ctrl, []*armoperationalinsights.Workspace{workspace1, workspace2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		workspace1 := createAzureWorkspace("test-workspace-1", subscriptionID, resourceGroup)
		workspaceNilName := &armoperationalinsights.Workspace{
			Name:     nil, // nil name should be skipped
			Location: new("eastus"),
			Tags: map[string]*string{
				"env": new("test"),
			},
		}

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockPager := newMockOperationalInsightsWorkspacePager(ctrl, []*armoperationalinsights.Workspace{workspace1, workspaceNilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("workspace not found")

		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-workspace", nil).Return(
			armoperationalinsights.WorkspacesClientGetResponse{}, expectedErr)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-workspace", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent workspace, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting workspace with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockOperationalInsightsWorkspaceClient(ctrl)

		wrapper := manual.NewOperationalInsightsWorkspace(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		// Test the wrapper's Get method directly with insufficient query parts
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when getting workspace with insufficient query parts, but got nil")
		}
	})
}

// createAzureWorkspace creates a mock Azure Log Analytics Workspace for testing
func createAzureWorkspace(workspaceName, subscriptionID, resourceGroup string) *armoperationalinsights.Workspace {
	succeededState := armoperationalinsights.WorkspaceEntityStatusSucceeded
	retentionDays := int32(30)
	return &armoperationalinsights.Workspace{
		Name:     new(workspaceName),
		Location: new("eastus"),
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OperationalInsights/workspaces/" + workspaceName),
		Type:     new("Microsoft.OperationalInsights/workspaces"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armoperationalinsights.WorkspaceProperties{
			ProvisioningState: &succeededState,
			RetentionInDays:   &retentionDays,
			Features: &armoperationalinsights.WorkspaceFeatures{
				ClusterResourceID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OperationalInsights/clusters/test-cluster"),
			},
			PrivateLinkScopedResources: []*armoperationalinsights.PrivateLinkScopedResource{
				{
					// Note: ResourceID refers to microsoft.insights/privateLinkScopes/scopedResources
					ResourceID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/microsoft.insights/privateLinkScopes/test-pls/scopedResources/test-scoped-resource"),
					ScopeID:    new("test-scope-id"),
				},
			},
		},
	}
}

// createAzureWorkspaceWithCrossResourceGroupLinks creates a mock Workspace with links to resources in different resource groups
func createAzureWorkspaceWithCrossResourceGroupLinks(workspaceName, subscriptionID string) *armoperationalinsights.Workspace {
	succeededState := armoperationalinsights.WorkspaceEntityStatusSucceeded
	return &armoperationalinsights.Workspace{
		Name:     new(workspaceName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armoperationalinsights.WorkspaceProperties{
			ProvisioningState: &succeededState,
			Features: &armoperationalinsights.WorkspaceFeatures{
				ClusterResourceID: new("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.OperationalInsights/clusters/test-cluster-cross-rg"),
			},
			PrivateLinkScopedResources: []*armoperationalinsights.PrivateLinkScopedResource{
				{
					ResourceID: new("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/microsoft.insights/privateLinkScopes/test-pls-cross/scopedResources/test-scoped-resource-cross"),
					ScopeID:    new("test-scope-id"),
				},
			},
		},
	}
}

// createAzureWorkspaceWithoutLinks creates a mock Workspace without any linked resources
func createAzureWorkspaceWithoutLinks(workspaceName string) *armoperationalinsights.Workspace {
	succeededState := armoperationalinsights.WorkspaceEntityStatusSucceeded
	return &armoperationalinsights.Workspace{
		Name:     new(workspaceName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armoperationalinsights.WorkspaceProperties{
			ProvisioningState: &succeededState,
			// No PrivateLinkScopedResources
		},
	}
}

// createAzureWorkspaceWithProvisioningState creates a mock Workspace with a specific provisioning state
func createAzureWorkspaceWithProvisioningState(workspaceName string, state armoperationalinsights.WorkspaceEntityStatus) *armoperationalinsights.Workspace {
	return &armoperationalinsights.Workspace{
		Name:     new(workspaceName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armoperationalinsights.WorkspaceProperties{
			ProvisioningState: &state,
		},
	}
}

// mockOperationalInsightsWorkspacePager is a simple mock implementation of the Pager interface for testing
type mockOperationalInsightsWorkspacePager struct {
	ctrl  *gomock.Controller
	items []*armoperationalinsights.Workspace
	index int
	more  bool
}

func newMockOperationalInsightsWorkspacePager(ctrl *gomock.Controller, items []*armoperationalinsights.Workspace) clients.OperationalInsightsWorkspacePager {
	return &mockOperationalInsightsWorkspacePager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockOperationalInsightsWorkspacePager) More() bool {
	return m.more
}

func (m *mockOperationalInsightsWorkspacePager) NextPage(ctx context.Context) (armoperationalinsights.WorkspacesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armoperationalinsights.WorkspacesClientListByResourceGroupResponse{
			WorkspaceListResult: armoperationalinsights.WorkspaceListResult{
				Value: []*armoperationalinsights.Workspace{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armoperationalinsights.WorkspacesClientListByResourceGroupResponse{
		WorkspaceListResult: armoperationalinsights.WorkspaceListResult{
			Value: []*armoperationalinsights.Workspace{item},
		},
	}, nil
}
