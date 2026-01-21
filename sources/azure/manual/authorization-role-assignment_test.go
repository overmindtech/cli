package manual_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestAuthorizationRoleAssignment(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		roleAssignmentName := "test-role-assignment"
		roleAssignment := createAzureRoleAssignment(roleAssignmentName, "/subscriptions/test-subscription/resourceGroups/test-rg")

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		azureScope := "/subscriptions/test-subscription/resourceGroups/test-rg"
		mockClient.EXPECT().Get(ctx, azureScope, roleAssignmentName, nil).Return(
			armauthorization.RoleAssignmentsClientGetResponse{
				RoleAssignment: *roleAssignment,
			}, nil)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, roleAssignmentName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.AuthorizationRoleAssignment.String() {
			t.Errorf("Expected type %s, got %s", azureshared.AuthorizationRoleAssignment.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(resourceGroup, roleAssignmentName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != scope {
			t.Errorf("Expected scope %s, got %s", scope, sdpItem.GetScope())
		}

		// Verify linked item queries
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Role Definition link
					ExpectedType:   azureshared.AuthorizationRoleDefinition.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "b24988ac-6180-42a0-ab88-20f7382dd24c",
					ExpectedScope:  subscriptionID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyScope", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, "", "test-role-assignment", true)
		if qErr == nil {
			t.Error("Expected error when getting role assignment with empty scope, but got nil")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (empty)
		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting role assignment with empty name, but got nil")
		}

		// Test with too many query parts - Get expects a single query string
		_, qErr = adapter.Get(ctx, scope, shared.CompositeLookupKey("name", "extra"), true)
		if qErr == nil {
			t.Error("Expected error when getting role assignment with too many query parts, but got nil")
		}
	})

	t.Run("Get_EmptyRoleAssignmentName", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting role assignment with empty name, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		roleAssignmentName := "test-role-assignment"
		expectedError := errors.New("client error")

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		azureScope := "/subscriptions/test-subscription/resourceGroups/test-rg"
		mockClient.EXPECT().Get(ctx, azureScope, roleAssignmentName, nil).Return(
			armauthorization.RoleAssignmentsClientGetResponse{},
			expectedError)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, roleAssignmentName, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Get_NilName", func(t *testing.T) {
		roleAssignment := &armauthorization.RoleAssignment{
			Name: nil, // Role assignment with nil name should cause error
			Properties: &armauthorization.RoleAssignmentProperties{
				Scope: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg"),
			},
		}

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		azureScope := "/subscriptions/test-subscription/resourceGroups/test-rg"
		roleAssignmentName := "test-role-assignment"
		mockClient.EXPECT().Get(ctx, azureScope, roleAssignmentName, nil).Return(
			armauthorization.RoleAssignmentsClientGetResponse{
				RoleAssignment: *roleAssignment,
			}, nil)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, roleAssignmentName, true)
		if qErr == nil {
			t.Error("Expected error when role assignment has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		roleAssignment1 := createAzureRoleAssignment("test-role-assignment-1", "/subscriptions/test-subscription/resourceGroups/test-rg")
		roleAssignment2 := createAzureRoleAssignment("test-role-assignment-2", "/subscriptions/test-subscription/resourceGroups/test-rg")

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		mockPager := NewMockRoleAssignmentsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleAssignmentsClientListForResourceGroupResponse{
					RoleAssignmentListResult: armauthorization.RoleAssignmentListResult{
						Value: []*armauthorization.RoleAssignment{roleAssignment1, roleAssignment2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().ListForResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
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

		for i, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			if item.GetType() != azureshared.AuthorizationRoleAssignment.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.AuthorizationRoleAssignment.String(), item.GetType())
			}

			expectedName := "test-role-assignment-" + string(rune(i+1+'0'))
			expectedUniqueAttrValue := shared.CompositeLookupKey(resourceGroup, expectedName)
			if item.UniqueAttributeValue() != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got: %s", expectedUniqueAttrValue, item.UniqueAttributeValue())
			}
		}
	})

	t.Run("List_EmptyScope", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, "", true)
		if err == nil {
			t.Error("Expected error when listing role assignments with empty scope, but got nil")
		}
	})

	t.Run("List_PagerError", func(t *testing.T) {
		expectedError := errors.New("pager error")

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		mockPager := NewMockRoleAssignmentsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleAssignmentsClientListForResourceGroupResponse{},
				expectedError),
		)

		mockClient.EXPECT().ListForResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
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
		// Create role assignment with nil name to test error handling
		roleAssignment1 := createAzureRoleAssignment("test-role-assignment-1", "/subscriptions/test-subscription/resourceGroups/test-rg")
		roleAssignment2 := &armauthorization.RoleAssignment{
			Name: nil, // Role assignment with nil name should cause error
			Properties: &armauthorization.RoleAssignmentProperties{
				Scope: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg"),
			},
		}

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		mockPager := NewMockRoleAssignmentsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armauthorization.RoleAssignmentsClientListForResourceGroupResponse{
					RoleAssignmentListResult: armauthorization.RoleAssignmentListResult{
						Value: []*armauthorization.RoleAssignment{roleAssignment1, roleAssignment2},
					},
				}, nil),
		)

		mockClient.EXPECT().ListForResourceGroup(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, scope, true)
		if err == nil {
			t.Error("Expected error when listing role assignments with nil name, but got nil")
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)

		lookups := wrapper.GetLookups()
		if len(lookups) != 1 {
			t.Errorf("Expected 1 lookup, got: %d", len(lookups))
		}

		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.AuthorizationRoleAssignment {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include AuthorizationRoleAssignment")
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)

		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}

		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_role_assignment.id" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod to be GET, got: %v", mapping.GetTerraformMethod())
				}
				break
			}
		}

		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_role_assignment.id' mapping")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)

		potentialLinks := wrapper.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to include at least one link type")
		}
		if !potentialLinks[azureshared.ManagedIdentityUserAssignedIdentity] {
			t.Error("Expected PotentialLinks to include ManagedIdentityUserAssignedIdentity")
		}
		if !potentialLinks[azureshared.AuthorizationRoleDefinition] {
			t.Error("Expected PotentialLinks to include AuthorizationRoleDefinition")
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)

		permissions := wrapper.IAMPermissions()
		if len(permissions) != 1 {
			t.Errorf("Expected 1 permission, got: %d", len(permissions))
		}

		expectedPermission := "Microsoft.Authorization/roleAssignments/read"
		if permissions[0] != expectedPermission {
			t.Errorf("Expected permission %s, got: %s", expectedPermission, permissions[0])
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)

		// Use interface assertion to access PredefinedRole method
		if roleInterface, ok := interface{}(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}
	})

	t.Run("Get_WithDelegatedManagedIdentity", func(t *testing.T) {
		roleAssignmentName := "test-role-assignment-with-identity"
		roleAssignment := createAzureRoleAssignment(roleAssignmentName, "/subscriptions/test-subscription/resourceGroups/test-rg")
		// Add delegated managed identity resource ID
		delegatedIdentityID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
		roleAssignment.Properties.DelegatedManagedIdentityResourceID = to.Ptr(delegatedIdentityID)

		mockClient := mocks.NewMockRoleAssignmentsClient(ctrl)
		azureScope := "/subscriptions/test-subscription/resourceGroups/test-rg"
		mockClient.EXPECT().Get(ctx, azureScope, roleAssignmentName, nil).Return(
			armauthorization.RoleAssignmentsClientGetResponse{
				RoleAssignment: *roleAssignment,
			}, nil)

		wrapper := manual.NewAuthorizationRoleAssignment(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, roleAssignmentName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify linked item queries include both role definition and managed identity
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Role Definition link
					ExpectedType:   azureshared.AuthorizationRoleDefinition.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "b24988ac-6180-42a0-ab88-20f7382dd24c",
					ExpectedScope:  subscriptionID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Delegated Managed Identity link
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity",
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})
}

// MockRoleAssignmentsPager is a mock for RoleAssignmentsPager
type MockRoleAssignmentsPager struct {
	ctrl     *gomock.Controller
	recorder *MockRoleAssignmentsPagerMockRecorder
}

type MockRoleAssignmentsPagerMockRecorder struct {
	mock *MockRoleAssignmentsPager
}

func NewMockRoleAssignmentsPager(ctrl *gomock.Controller) *MockRoleAssignmentsPager {
	mock := &MockRoleAssignmentsPager{ctrl: ctrl}
	mock.recorder = &MockRoleAssignmentsPagerMockRecorder{mock}
	return mock
}

func (m *MockRoleAssignmentsPager) EXPECT() *MockRoleAssignmentsPagerMockRecorder {
	return m.recorder
}

func (m *MockRoleAssignmentsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockRoleAssignmentsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockRoleAssignmentsPager)(nil).More))
}

func (m *MockRoleAssignmentsPager) NextPage(ctx context.Context) (armauthorization.RoleAssignmentsClientListForResourceGroupResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armauthorization.RoleAssignmentsClientListForResourceGroupResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockRoleAssignmentsPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockRoleAssignmentsPager)(nil).NextPage), ctx)
}

// createAzureRoleAssignment creates a mock Azure role assignment for testing
func createAzureRoleAssignment(roleAssignmentName, scope string) *armauthorization.RoleAssignment {
	return &armauthorization.RoleAssignment{
		Name: to.Ptr(roleAssignmentName),
		Type: to.Ptr("Microsoft.Authorization/roleAssignments"),
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Authorization/roleAssignments/" + roleAssignmentName),
		Properties: &armauthorization.RoleAssignmentProperties{
			Scope:            to.Ptr(scope),
			RoleDefinitionID: to.Ptr("/subscriptions/test-subscription/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c"),
			PrincipalID:      to.Ptr("00000000-0000-0000-0000-000000000000"),
		},
	}
}
