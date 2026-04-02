package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockFederatedIdentityCredentialsPager is a simple mock implementation of FederatedIdentityCredentialsPager
type mockFederatedIdentityCredentialsPager struct {
	pages []armmsi.FederatedIdentityCredentialsClientListResponse
	index int
}

func (m *mockFederatedIdentityCredentialsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockFederatedIdentityCredentialsPager) NextPage(ctx context.Context) (armmsi.FederatedIdentityCredentialsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armmsi.FederatedIdentityCredentialsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorFederatedIdentityCredentialsPager is a mock pager that always returns an error
type errorFederatedIdentityCredentialsPager struct{}

func (e *errorFederatedIdentityCredentialsPager) More() bool {
	return true
}

func (e *errorFederatedIdentityCredentialsPager) NextPage(ctx context.Context) (armmsi.FederatedIdentityCredentialsClientListResponse, error) {
	return armmsi.FederatedIdentityCredentialsClientListResponse{}, errors.New("pager error")
}

// testFederatedIdentityCredentialsClient wraps the mock to implement the correct interface
type testFederatedIdentityCredentialsClient struct {
	*mocks.MockFederatedIdentityCredentialsClient
	pager clients.FederatedIdentityCredentialsPager
}

func (t *testFederatedIdentityCredentialsClient) NewListPager(resourceGroupName string, resourceName string, options *armmsi.FederatedIdentityCredentialsClientListOptions) clients.FederatedIdentityCredentialsPager {
	return t.pager
}

func TestManagedIdentityFederatedIdentityCredential(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	identityName := "test-identity"
	credentialName := "test-credential"

	t.Run("Get", func(t *testing.T) {
		credential := createAzureFederatedIdentityCredential(credentialName)

		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, identityName, credentialName, nil).Return(
			armmsi.FederatedIdentityCredentialsClientGetResponse{
				FederatedIdentityCredential: *credential,
			}, nil)

		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}
		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(identityName, credentialName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ManagedIdentityFederatedIdentityCredential.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ManagedIdentityFederatedIdentityCredential, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(identityName, credentialName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(identityName, credentialName), sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) != 2 {
				t.Fatalf("Expected 2 linked queries, got: %d", len(linkedQueries))
			}

			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  identityName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "token.actions.githubusercontent.com",
					ExpectedScope:  "global",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], identityName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyIdentityName", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", credentialName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting with empty identity name, but got nil")
		}
	})

	t.Run("GetWithEmptyCredentialName", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(identityName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting with empty credential name, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		credential1 := createAzureFederatedIdentityCredential("credential-1")
		credential2 := createAzureFederatedIdentityCredential("credential-2")

		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		mockPager := &mockFederatedIdentityCredentialsPager{
			pages: []armmsi.FederatedIdentityCredentialsClientListResponse{
				{
					FederatedIdentityCredentialsListResult: armmsi.FederatedIdentityCredentialsListResult{
						Value: []*armmsi.FederatedIdentityCredential{credential1, credential2},
					},
				},
			},
		}

		testClient := &testFederatedIdentityCredentialsClient{
			MockFederatedIdentityCredentialsClient: mockClient,
			pager:                                  mockPager,
		}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], identityName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}

			if item.GetType() != azureshared.ManagedIdentityFederatedIdentityCredential.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ManagedIdentityFederatedIdentityCredential, item.GetType())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		credential1 := createAzureFederatedIdentityCredential("credential-1")
		credential2 := createAzureFederatedIdentityCredential("credential-2")

		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		mockPager := &mockFederatedIdentityCredentialsPager{
			pages: []armmsi.FederatedIdentityCredentialsClientListResponse{
				{
					FederatedIdentityCredentialsListResult: armmsi.FederatedIdentityCredentialsListResult{
						Value: []*armmsi.FederatedIdentityCredential{credential1, credential2},
					},
				},
			},
		}

		testClient := &testFederatedIdentityCredentialsClient{
			MockFederatedIdentityCredentialsClient: mockClient,
			pager:                                  mockPager,
		}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], identityName, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("SearchWithEmptyIdentityName", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when providing empty identity name, but got nil")
		}
	})

	t.Run("SearchWithNoQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_CredentialWithNilName", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		mockPager := &mockFederatedIdentityCredentialsPager{
			pages: []armmsi.FederatedIdentityCredentialsClientListResponse{
				{
					FederatedIdentityCredentialsListResult: armmsi.FederatedIdentityCredentialsListResult{
						Value: []*armmsi.FederatedIdentityCredential{
							{Name: nil},
							createAzureFederatedIdentityCredential("valid-credential"),
						},
					},
				},
			},
		}

		testClient := &testFederatedIdentityCredentialsClient{
			MockFederatedIdentityCredentialsClient: mockClient,
			pager:                                  mockPager,
		}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], identityName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(identityName, "valid-credential") {
			t.Errorf("Expected credential unique value '%s', got %s", shared.CompositeLookupKey(identityName, "valid-credential"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("credential not found")

		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, identityName, "nonexistent", nil).Return(
			armmsi.FederatedIdentityCredentialsClientGetResponse{}, expectedErr)

		testClient := &testFederatedIdentityCredentialsClient{MockFederatedIdentityCredentialsClient: mockClient}
		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(identityName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent credential, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockFederatedIdentityCredentialsClient(ctrl)
		errorPager := &errorFederatedIdentityCredentialsPager{}

		testClient := &testFederatedIdentityCredentialsClient{
			MockFederatedIdentityCredentialsClient: mockClient,
			pager:                                  errorPager,
		}

		wrapper := manual.NewManagedIdentityFederatedIdentityCredential(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], identityName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureFederatedIdentityCredential(name string) *armmsi.FederatedIdentityCredential {
	return &armmsi.FederatedIdentityCredential{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity/federatedIdentityCredentials/" + name),
		Name: new(name),
		Type: new("Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials"),
		Properties: &armmsi.FederatedIdentityCredentialProperties{
			Issuer:    new("https://token.actions.githubusercontent.com"),
			Subject:   new("repo:example/repo:ref:refs/heads/main"),
			Audiences: []*string{new("api://AzureADTokenExchange")},
		},
	}
}
