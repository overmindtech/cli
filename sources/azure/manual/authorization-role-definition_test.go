package manual_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestAuthorizationRoleDefinition(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	scope := subscriptionID
	azureScope := "/subscriptions/" + subscriptionID

	t.Run("Get", func(t *testing.T) {
		roleDefinitionID := "b24988ac-6180-42a0-ab88-20f7382dd24c"
		roleDefinition := createAzureRoleDefinition(roleDefinitionID, "Reader")

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, azureScope, roleDefinitionID, nil).Return(
			armauthorization.RoleDefinitionsClientGetResponse{
				RoleDefinition: *roleDefinition,
			}, nil)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, roleDefinitionID, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.AuthorizationRoleDefinition.String() {
			t.Errorf("Expected type %s, got %s", azureshared.AuthorizationRoleDefinition.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != roleDefinitionID {
			t.Errorf("Expected unique attribute value %s, got %s", roleDefinitionID, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != scope {
			t.Errorf("Expected scope %s, got %s", scope, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		// Verify linked item queries for AssignableScopes
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Subscription scope link
					ExpectedType:   azureshared.ResourcesSubscription.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  subscriptionID,
					ExpectedScope:  "global",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyScope", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, "", "test-role-definition", true)
		if qErr == nil {
			t.Error("Expected error when getting role definition with empty scope, but got nil")
		}
	})

	t.Run("Get_EmptyRoleDefinitionID", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting role definition with empty ID, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		roleDefinitionID := "test-role-definition"
		expectedError := errors.New("client error")

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, azureScope, roleDefinitionID, nil).Return(
			armauthorization.RoleDefinitionsClientGetResponse{},
			expectedError)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, roleDefinitionID, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Get_NilName", func(t *testing.T) {
		roleDefinition := &armauthorization.RoleDefinition{
			Name: nil,
			Properties: &armauthorization.RoleDefinitionProperties{
				RoleName: new("Reader"),
			},
		}

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		roleDefinitionID := "test-role-definition"
		mockClient.EXPECT().Get(ctx, azureScope, roleDefinitionID, nil).Return(
			armauthorization.RoleDefinitionsClientGetResponse{
				RoleDefinition: *roleDefinition,
			}, nil)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, roleDefinitionID, true)
		if qErr == nil {
			t.Error("Expected error when role definition has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		roleDefinition1 := createAzureRoleDefinition("guid-1", "Reader")
		roleDefinition2 := createAzureRoleDefinition("guid-2", "Contributor")

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockPager := NewMockRoleDefinitionsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleDefinitionsClientListResponse{
					RoleDefinitionListResult: armauthorization.RoleDefinitionListResult{
						Value: []*armauthorization.RoleDefinition{roleDefinition1, roleDefinition2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(azureScope, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
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

			if item.GetType() != azureshared.AuthorizationRoleDefinition.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.AuthorizationRoleDefinition.String(), item.GetType())
			}
		}
	})

	t.Run("List_EmptyScope", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, "", true)
		if err == nil {
			t.Error("Expected error when listing role definitions with empty scope, but got nil")
		}
	})

	t.Run("List_PagerError", func(t *testing.T) {
		expectedError := errors.New("pager error")

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockPager := NewMockRoleDefinitionsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleDefinitionsClientListResponse{},
				expectedError),
		)

		mockClient.EXPECT().NewListPager(azureScope, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, scope, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		roleDefinition1 := createAzureRoleDefinition("guid-1", "Reader")
		roleDefinition2 := &armauthorization.RoleDefinition{
			Name: nil,
			Properties: &armauthorization.RoleDefinitionProperties{
				RoleName: new("Contributor"),
			},
		}

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockPager := NewMockRoleDefinitionsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleDefinitionsClientListResponse{
					RoleDefinitionListResult: armauthorization.RoleDefinitionListResult{
						Value: []*armauthorization.RoleDefinition{roleDefinition1, roleDefinition2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(azureScope, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should skip nil name items
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		roleDefinition1 := createAzureRoleDefinition("guid-1", "Reader")
		roleDefinition2 := createAzureRoleDefinition("guid-2", "Contributor")

		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		mockPager := NewMockRoleDefinitionsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleDefinitionsClientListResponse{
					RoleDefinitionListResult: armauthorization.RoleDefinitionListResult{
						Value: []*armauthorization.RoleDefinition{roleDefinition1, roleDefinition2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(azureScope, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

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

		listStreamable.ListStream(ctx, scope, true, stream)
		wg.Wait()

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %d", len(errs))
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)

		lookups := wrapper.GetLookups()
		if len(lookups) != 1 {
			t.Errorf("Expected 1 lookup, got: %d", len(lookups))
		}

		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.AuthorizationRoleDefinition {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include AuthorizationRoleDefinition")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)

		potentialLinks := wrapper.PotentialLinks()
		if len(potentialLinks) != 2 {
			t.Errorf("Expected 2 potential links, got: %d", len(potentialLinks))
		}
		if !potentialLinks[azureshared.ResourcesSubscription] {
			t.Error("Expected PotentialLinks to include ResourcesSubscription")
		}
		if !potentialLinks[azureshared.ResourcesResourceGroup] {
			t.Error("Expected PotentialLinks to include ResourcesResourceGroup")
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)

		permissions := wrapper.IAMPermissions()
		if len(permissions) != 1 {
			t.Errorf("Expected 1 permission, got: %d", len(permissions))
		}

		expectedPermission := "Microsoft.Authorization/roleDefinitions/read"
		if permissions[0] != expectedPermission {
			t.Errorf("Expected permission %s, got: %s", expectedPermission, permissions[0])
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockRoleDefinitionsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleDefinition(mockClient, subscriptionID)

		if roleInterface, ok := any(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}
	})
}

// MockRoleDefinitionsPager is a mock for RoleDefinitionsPager
type MockRoleDefinitionsPager struct {
	ctrl     *gomock.Controller
	recorder *MockRoleDefinitionsPagerMockRecorder
}

type MockRoleDefinitionsPagerMockRecorder struct {
	mock *MockRoleDefinitionsPager
}

func NewMockRoleDefinitionsPager(ctrl *gomock.Controller) *MockRoleDefinitionsPager {
	mock := &MockRoleDefinitionsPager{ctrl: ctrl}
	mock.recorder = &MockRoleDefinitionsPagerMockRecorder{mock}
	return mock
}

func (m *MockRoleDefinitionsPager) EXPECT() *MockRoleDefinitionsPagerMockRecorder {
	return m.recorder
}

func (m *MockRoleDefinitionsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockRoleDefinitionsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeFor[func() bool]())
}

func (m *MockRoleDefinitionsPager) NextPage(ctx context.Context) (armauthorization.RoleDefinitionsClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armauthorization.RoleDefinitionsClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockRoleDefinitionsPagerMockRecorder) NextPage(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeFor[func(ctx context.Context) (armauthorization.RoleDefinitionsClientListResponse, error)](), ctx)
}

// createAzureRoleDefinition creates a mock Azure role definition for testing
func createAzureRoleDefinition(roleDefinitionID, roleName string) *armauthorization.RoleDefinition {
	return &armauthorization.RoleDefinition{
		Name: new(roleDefinitionID),
		Type: new("Microsoft.Authorization/roleDefinitions"),
		ID:   new("/subscriptions/test-subscription/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionID),
		Properties: &armauthorization.RoleDefinitionProperties{
			RoleName:    new(roleName),
			RoleType:    new("BuiltInRole"),
			Description: new("Test role definition for " + roleName),
			AssignableScopes: []*string{
				new("/subscriptions/test-subscription"),
			},
			Permissions: []*armauthorization.Permission{
				{
					Actions: []*string{
						new("*/read"),
					},
				},
			},
		},
	}
}
