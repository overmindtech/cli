package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockManagedHSMsPager is a simple mock implementation of ManagedHSMsPager
type mockManagedHSMsPager struct {
	pages []armkeyvault.ManagedHsmsClientListByResourceGroupResponse
	index int
}

func (m *mockManagedHSMsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockManagedHSMsPager) NextPage(ctx context.Context) (armkeyvault.ManagedHsmsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armkeyvault.ManagedHsmsClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorManagedHSMsPager is a mock pager that always returns an error
type errorManagedHSMsPager struct{}

func (e *errorManagedHSMsPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorManagedHSMsPager) NextPage(ctx context.Context) (armkeyvault.ManagedHsmsClientListByResourceGroupResponse, error) {
	return armkeyvault.ManagedHsmsClientListByResourceGroupResponse{}, errors.New("pager error")
}

// testManagedHSMsClient wraps the mock to implement the correct interface
type testManagedHSMsClient struct {
	*mocks.MockManagedHSMsClient
	pager clients.ManagedHSMsPager
}

func (t *testManagedHSMsClient) NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.ManagedHsmsClientListByResourceGroupOptions) clients.ManagedHSMsPager {
	// Call the mock to satisfy expectations
	t.MockManagedHSMsClient.NewListByResourceGroupPager(resourceGroupName, options)
	return t.pager
}

func TestKeyVaultManagedHSM(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	hsmName := "test-managed-hsm"

	t.Run("Get", func(t *testing.T) {
		hsm := createAzureManagedHSM(hsmName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, nil).Return(
			armkeyvault.ManagedHsmsClientGetResponse{
				ManagedHsm: *hsm,
			}, nil)

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.KeyVaultManagedHSM.String() {
			t.Errorf("Expected type %s, got %s", azureshared.KeyVaultManagedHSM, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != hsmName {
			t.Errorf("Expected unique attribute value %s, got %s", hsmName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Private Endpoint (GET) - same resource group
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Private Endpoint (GET) - different resource group
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint-diff-rg",
					ExpectedScope:  subscriptionID + ".different-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Subnet (GET) - same resource group
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Subnet (GET) - different resource group
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet-diff-rg", "test-subnet-diff-rg"),
					ExpectedScope:  subscriptionID + ".different-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User Assigned Managed Identity (GET) - same resource group
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User Assigned Managed Identity (GET) - different resource group
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity-diff-rg",
					ExpectedScope:  subscriptionID + ".identity-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DNS (SEARCH) - from HsmURI
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  hsmName + ".managedhsm.azure.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// IP (GET) - from NetworkACLs IPRules
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// IP (GET) - from NetworkACLs IPRules (CIDR range)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.0/24",
					ExpectedScope:  "global",
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
		mockClient := mocks.NewMockManagedHSMsClient(ctrl)

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with empty name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting Managed HSM with empty name, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		hsm := &armkeyvault.ManagedHsm{
			Name: nil, // No name field
			Properties: &armkeyvault.ManagedHsmProperties{
				TenantID: to.Ptr("test-tenant-id"),
			},
		}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, nil).Return(
			armkeyvault.ManagedHsmsClientGetResponse{
				ManagedHsm: *hsm,
			}, nil)

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr == nil {
			t.Error("Expected error when Managed HSM has no name, but got nil")
		}
	})

	t.Run("Get_NoLinkedResources", func(t *testing.T) {
		hsm := createAzureManagedHSMMinimal(hsmName)

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, nil).Return(
			armkeyvault.ManagedHsmsClientGetResponse{
				ManagedHsm: *hsm,
			}, nil)

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have no linked item queries
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked item queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		hsm1 := createAzureManagedHSM("test-managed-hsm-1", subscriptionID, resourceGroup)
		hsm2 := createAzureManagedHSM("test-managed-hsm-2", subscriptionID, resourceGroup)

		mockPager := &mockManagedHSMsPager{
			pages: []armkeyvault.ManagedHsmsClientListByResourceGroupResponse{
				{
					ManagedHsmListResult: armkeyvault.ManagedHsmListResult{
						Value: []*armkeyvault.ManagedHsm{hsm1, hsm2},
					},
				},
			},
		}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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

		// Verify first item
		if sdpItems[0].UniqueAttributeValue() != "test-managed-hsm-1" {
			t.Errorf("Expected first item name 'test-managed-hsm-1', got %s", sdpItems[0].UniqueAttributeValue())
		}

		// Verify second item
		if sdpItems[1].UniqueAttributeValue() != "test-managed-hsm-2" {
			t.Errorf("Expected second item name 'test-managed-hsm-2', got %s", sdpItems[1].UniqueAttributeValue())
		}
	})

	t.Run("List_Error", func(t *testing.T) {
		errorPager := &errorManagedHSMsPager{}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 errorPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("List_SkipNilName", func(t *testing.T) {
		hsm1 := createAzureManagedHSM("test-managed-hsm-1", subscriptionID, resourceGroup)
		hsm2 := &armkeyvault.ManagedHsm{
			Name: nil, // This should be skipped
			Properties: &armkeyvault.ManagedHsmProperties{
				TenantID: to.Ptr("test-tenant-id"),
			},
		}
		hsm3 := createAzureManagedHSM("test-managed-hsm-3", subscriptionID, resourceGroup)

		mockPager := &mockManagedHSMsPager{
			pages: []armkeyvault.ManagedHsmsClientListByResourceGroupResponse{
				{
					ManagedHsmListResult: armkeyvault.ManagedHsmListResult{
						Value: []*armkeyvault.ManagedHsm{hsm1, hsm2, hsm3},
					},
				},
			},
		}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only have 2 items (hsm2 with nil name should be skipped)
		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items (skipping nil name), got: %d", len(sdpItems))
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		hsm1 := createAzureManagedHSM("test-managed-hsm-1", subscriptionID, resourceGroup)
		hsm2 := createAzureManagedHSM("test-managed-hsm-2", subscriptionID, resourceGroup)

		mockPager := &mockManagedHSMsPager{
			pages: []armkeyvault.ManagedHsmsClientListByResourceGroupResponse{
				{
					ManagedHsmListResult: armkeyvault.ManagedHsmListResult{
						Value: []*armkeyvault.ManagedHsm{hsm1, hsm2},
					},
				},
			},
		}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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

		// Verify first item
		if items[0].UniqueAttributeValue() != "test-managed-hsm-1" {
			t.Errorf("Expected first item name 'test-managed-hsm-1', got %s", items[0].UniqueAttributeValue())
		}

		// Verify second item
		if items[1].UniqueAttributeValue() != "test-managed-hsm-2" {
			t.Errorf("Expected second item name 'test-managed-hsm-2', got %s", items[1].UniqueAttributeValue())
		}

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream_Error", func(t *testing.T) {
		errorPager := &errorManagedHSMsPager{}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 errorPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(func(item *sdp.Item) {}, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)

		if len(errs) == 0 {
			t.Error("Expected error when pager returns error, but got none")
		}
	})

	t.Run("ListStream_SkipNilName", func(t *testing.T) {
		hsm1 := createAzureManagedHSM("test-managed-hsm-1", subscriptionID, resourceGroup)
		hsm2 := &armkeyvault.ManagedHsm{
			Name: nil, // This should be skipped
			Properties: &armkeyvault.ManagedHsmProperties{
				TenantID: to.Ptr("test-tenant-id"),
			},
		}
		hsm3 := createAzureManagedHSM("test-managed-hsm-3", subscriptionID, resourceGroup)

		mockPager := &mockManagedHSMsPager{
			pages: []armkeyvault.ManagedHsmsClientListByResourceGroupResponse{
				{
					ManagedHsmListResult: armkeyvault.ManagedHsmListResult{
						Value: []*armkeyvault.ManagedHsm{hsm1, hsm2, hsm3},
					},
				},
			},
		}

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		testClient := &testManagedHSMsClient{
			MockManagedHSMsClient: mockClient,
			pager:                 mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSM(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		wg := &sync.WaitGroup{}
		wg.Add(2) // we expect two items (hsm2 with nil name should be skipped)

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

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		// Should only have 2 items (hsm2 with nil name should be skipped)
		if len(items) != 2 {
			t.Fatalf("Expected 2 items (skipping nil name), got: %d", len(items))
		}

		// Verify items
		if items[0].UniqueAttributeValue() != "test-managed-hsm-1" {
			t.Errorf("Expected first item name 'test-managed-hsm-1', got %s", items[0].UniqueAttributeValue())
		}

		if items[1].UniqueAttributeValue() != "test-managed-hsm-3" {
			t.Errorf("Expected second item name 'test-managed-hsm-3', got %s", items[1].UniqueAttributeValue())
		}
	})

	t.Run("Get_Error", func(t *testing.T) {
		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, nil).Return(
			armkeyvault.ManagedHsmsClientGetResponse{},
			errors.New("client error"))

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("CrossResourceGroupScopes", func(t *testing.T) {
		// Test that linked resources in different resource groups use correct scopes
		hsm := createAzureManagedHSMCrossRG(hsmName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockManagedHSMsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, nil).Return(
			armkeyvault.ManagedHsmsClientGetResponse{
				ManagedHsm: *hsm,
			}, nil)

		wrapper := manual.NewKeyVaultManagedHSM(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that linked resources use their own scopes, not the Managed HSM's scope
		foundDifferentScope := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			scope := linkedQuery.GetQuery().GetScope()
			if scope != subscriptionID+"."+resourceGroup {
				foundDifferentScope = true
				// Verify the scope format is correct
				if scope != subscriptionID+".different-rg" && scope != subscriptionID+".identity-rg" {
					t.Errorf("Unexpected scope format: %s", scope)
				}
			}
		}

		if !foundDifferentScope {
			t.Error("Expected to find at least one linked item query with a different scope, but all used default scope")
		}
	})
}

// createAzureManagedHSM creates a mock Azure Managed HSM with linked resources
func createAzureManagedHSM(hsmName, subscriptionID, resourceGroup string) *armkeyvault.ManagedHsm {
	return &armkeyvault.ManagedHsm{
		Name:     to.Ptr(hsmName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armkeyvault.ManagedHsmProperties{
			TenantID: to.Ptr("test-tenant-id"),
			HsmURI:   to.Ptr("https://" + hsmName + ".managedhsm.azure.net"),
			// Private Endpoint Connections
			PrivateEndpointConnections: []*armkeyvault.MHSMPrivateEndpointConnectionItem{
				{
					Properties: &armkeyvault.MHSMPrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.MHSMPrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint"),
						},
					},
				},
				{
					Properties: &armkeyvault.MHSMPrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.MHSMPrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-diff-rg"),
						},
					},
				},
			},
			// Network ACLs with Virtual Network Rules and IP Rules
			NetworkACLs: &armkeyvault.MHSMNetworkRuleSet{
				VirtualNetworkRules: []*armkeyvault.MHSMVirtualNetworkRule{
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet-diff-rg/subnets/test-subnet-diff-rg"),
					},
				},
				IPRules: []*armkeyvault.MHSMIPRule{
					{
						Value: to.Ptr("192.168.1.1"),
					},
					{
						Value: to.Ptr("10.0.0.0/24"),
					},
				},
			},
		},
		// User Assigned Identities
		Identity: &armkeyvault.ManagedServiceIdentity{
			Type: to.Ptr(armkeyvault.ManagedServiceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armkeyvault.UserAssignedIdentity{
				"/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
				"/subscriptions/" + subscriptionID + "/resourceGroups/identity-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity-diff-rg":   {},
			},
		},
	}
}

// createAzureManagedHSMMinimal creates a minimal mock Azure Managed HSM without linked resources
func createAzureManagedHSMMinimal(hsmName string) *armkeyvault.ManagedHsm {
	return &armkeyvault.ManagedHsm{
		Name:     to.Ptr(hsmName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.ManagedHsmProperties{
			TenantID: to.Ptr("test-tenant-id"),
		},
	}
}

// createAzureManagedHSMCrossRG creates a mock Azure Managed HSM with linked resources in different resource groups
func createAzureManagedHSMCrossRG(hsmName, subscriptionID, resourceGroup string) *armkeyvault.ManagedHsm {
	return &armkeyvault.ManagedHsm{
		Name:     to.Ptr(hsmName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.ManagedHsmProperties{
			TenantID: to.Ptr("test-tenant-id"),
			// Private Endpoint in different resource group
			PrivateEndpointConnections: []*armkeyvault.MHSMPrivateEndpointConnectionItem{
				{
					Properties: &armkeyvault.MHSMPrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.MHSMPrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-pe-diff-rg"),
						},
					},
				},
			},
			// Subnet in different resource group
			NetworkACLs: &armkeyvault.MHSMNetworkRuleSet{
				VirtualNetworkRules: []*armkeyvault.MHSMVirtualNetworkRule{
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
				},
			},
		},
		// User Assigned Identity in different resource group
		Identity: &armkeyvault.ManagedServiceIdentity{
			Type: to.Ptr(armkeyvault.ManagedServiceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armkeyvault.UserAssignedIdentity{
				"/subscriptions/" + subscriptionID + "/resourceGroups/identity-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity-diff-rg": {},
			},
		},
	}
}
