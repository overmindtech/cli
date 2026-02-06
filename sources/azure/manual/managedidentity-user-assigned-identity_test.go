package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestManagedIdentityUserAssignedIdentity(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		identityName := "test-identity"
		identity := createAzureUserAssignedIdentity(identityName)

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, identityName, nil).Return(
			armmsi.UserAssignedIdentitiesClientGetResponse{
				Identity: *identity,
			}, nil)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], identityName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ManagedIdentityUserAssignedIdentity.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ManagedIdentityUserAssignedIdentity.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != identityName {
			t.Errorf("Expected unique attribute value %s, got %s", identityName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Federated identity credentials link
					ExpectedType:   azureshared.ManagedIdentityFederatedIdentityCredential.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  identityName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting user assigned identity with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		identity1 := createAzureUserAssignedIdentity("test-identity-1")
		identity2 := createAzureUserAssignedIdentity("test-identity-2")

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockPager := newMockUserAssignedIdentitiesPager(ctrl, []*armmsi.Identity{identity1, identity2})

		mockClient.EXPECT().ListByResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.ManagedIdentityUserAssignedIdentity.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.ManagedIdentityUserAssignedIdentity.String(), item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create identity with nil name to test filtering
		identity1 := createAzureUserAssignedIdentity("test-identity-1")
		identity2 := &armmsi.Identity{
			Name:     nil, // Identity with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armmsi.UserAssignedIdentityProperties{
				ClientID:    to.Ptr("test-client-id-2"),
				PrincipalID: to.Ptr("test-principal-id-2"),
				TenantID:    to.Ptr("test-tenant-id"),
			},
		}

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockPager := newMockUserAssignedIdentitiesPager(ctrl, []*armmsi.Identity{identity1, identity2})

		mockClient.EXPECT().ListByResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (identity with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "test-identity-1" {
			t.Fatalf("Expected identity name 'test-identity-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		identity1 := createAzureUserAssignedIdentity("test-identity-1")
		identity2 := createAzureUserAssignedIdentity("test-identity-2")

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockPager := newMockUserAssignedIdentitiesPager(ctrl, []*armmsi.Identity{identity1, identity2})

		mockClient.EXPECT().ListByResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		// Check if adapter supports list streaming
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

	t.Run("ListStream_ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("failed to list user assigned identities")

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockPager := newErrorUserAssignedIdentitiesPager(ctrl, expectedErr)

		mockClient.EXPECT().ListByResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(func(*sdp.Item) {}, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)

		if len(errs) == 0 {
			t.Error("Expected error when listing user assigned identities fails, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("user assigned identity not found")

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-identity", nil).Return(
			armmsi.UserAssignedIdentitiesClientGetResponse{}, expectedErr)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-identity", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent user assigned identity, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list user assigned identities")

		mockClient := mocks.NewMockUserAssignedIdentitiesClient(ctrl)
		mockPager := newErrorUserAssignedIdentitiesPager(ctrl, expectedErr)

		mockClient.EXPECT().ListByResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewManagedIdentityUserAssignedIdentity(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing user assigned identities fails, but got nil")
		}
	})
}

// createAzureUserAssignedIdentity creates a mock Azure User Assigned Identity for testing
func createAzureUserAssignedIdentity(identityName string) *armmsi.Identity {
	return &armmsi.Identity{
		Name:     to.Ptr(identityName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armmsi.UserAssignedIdentityProperties{
			ClientID:    to.Ptr("test-client-id"),
			PrincipalID: to.Ptr("test-principal-id"),
			TenantID:    to.Ptr("test-tenant-id"),
		},
	}
}

// mockUserAssignedIdentitiesPager is a simple mock implementation of UserAssignedIdentitiesPager
type mockUserAssignedIdentitiesPager struct {
	ctrl  *gomock.Controller
	items []*armmsi.Identity
	index int
	more  bool
}

func newMockUserAssignedIdentitiesPager(ctrl *gomock.Controller, items []*armmsi.Identity) clients.UserAssignedIdentitiesPager {
	return &mockUserAssignedIdentitiesPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockUserAssignedIdentitiesPager) More() bool {
	return m.more
}

func (m *mockUserAssignedIdentitiesPager) NextPage(ctx context.Context) (armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{
			UserAssignedIdentitiesListResult: armmsi.UserAssignedIdentitiesListResult{
				Value: []*armmsi.Identity{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{
		UserAssignedIdentitiesListResult: armmsi.UserAssignedIdentitiesListResult{
			Value: []*armmsi.Identity{item},
		},
	}, nil
}

// errorUserAssignedIdentitiesPager is a mock pager that returns an error on NextPage
type errorUserAssignedIdentitiesPager struct {
	ctrl *gomock.Controller
	err  error
	more bool
}

func newErrorUserAssignedIdentitiesPager(ctrl *gomock.Controller, err error) clients.UserAssignedIdentitiesPager {
	return &errorUserAssignedIdentitiesPager{
		ctrl: ctrl,
		err:  err,
		more: true, // Return true initially so NextPage will be called
	}
}

func (e *errorUserAssignedIdentitiesPager) More() bool {
	return e.more
}

func (e *errorUserAssignedIdentitiesPager) NextPage(ctx context.Context) (armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse, error) {
	e.more = false // After returning error, More() should return false to stop the loop
	return armmsi.UserAssignedIdentitiesClientListByResourceGroupResponse{}, e.err
}
