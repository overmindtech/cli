package manual_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
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

func TestComputeDiskEncryptionSet(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		desName := "test-des"
		des := createAzureDiskEncryptionSet(desName)

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, desName, nil).Return(
			armcompute.DiskEncryptionSetsClientGetResponse{DiskEncryptionSet: *des},
			nil,
		)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], desName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDiskEncryptionSet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDiskEncryptionSet.String(), sdpItem.GetType())
		}
		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}
		if sdpItem.UniqueAttributeValue() != desName {
			t.Errorf("Expected unique attribute value %s, got %s", desName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}
	})

	t.Run("GetWithAllLinkedResources", func(t *testing.T) {
		desName := "test-des"
		des := createAzureDiskEncryptionSetWithAllLinks(desName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, desName, nil).Return(
			armcompute.DiskEncryptionSetsClientGetResponse{DiskEncryptionSet: *des},
			nil,
		)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], desName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				// Properties.ActiveKey.SourceVault.ID - Key Vault Vault
				ExpectedType:   azureshared.KeyVaultVault.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-vault",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.ActiveKey.KeyURL - Key Vault Key
				ExpectedType:   azureshared.KeyVaultKey.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  shared.CompositeLookupKey("test-vault", "test-key"),
				ExpectedScope:  subscriptionID + "." + resourceGroup, // Key Vault URI doesn't contain resource group, use DES scope
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.ActiveKey.KeyURL - DNS name
				ExpectedType:   "dns",
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "test-vault.vault.azure.net",
				ExpectedScope:  "global",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
			{
				// Identity.UserAssignedIdentities[{id}] - User Assigned Identity
				ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-identity",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		}

		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})

	t.Run("GetWithPreviousKeysLinks", func(t *testing.T) {
		desName := "test-des"
		des := createAzureDiskEncryptionSetWithPreviousKeys(desName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, desName, nil).Return(
			armcompute.DiskEncryptionSetsClientGetResponse{DiskEncryptionSet: *des},
			nil,
		)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], desName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				// Properties.ActiveKey.SourceVault.ID - Key Vault Vault
				ExpectedType:   azureshared.KeyVaultVault.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-vault",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.ActiveKey.KeyURL - Key Vault Key
				ExpectedType:   azureshared.KeyVaultKey.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  shared.CompositeLookupKey("test-vault", "test-key"),
				ExpectedScope:  subscriptionID + "." + resourceGroup, // Key Vault URI doesn't contain resource group, use DES scope
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.ActiveKey.KeyURL - DNS name
				ExpectedType:   "dns",
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "test-vault.vault.azure.net",
				ExpectedScope:  "global",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
			{
				// Identity.UserAssignedIdentities[{id}] - User Assigned Identity
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
				// Properties.PreviousKeys[].SourceVault.ID - Key Vault Vault
				ExpectedType:   azureshared.KeyVaultVault.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-old-vault",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.PreviousKeys[].KeyURL - Key Vault Key
				ExpectedType:   azureshared.KeyVaultKey.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  shared.CompositeLookupKey("test-old-vault", "test-old-key"),
				ExpectedScope:  subscriptionID + "." + resourceGroup, // Key Vault URI doesn't contain resource group, use DES scope
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			{
				// Properties.PreviousKeys[].KeyURL - DNS name
				ExpectedType:   "dns",
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "test-old-vault.vault.azure.net",
				ExpectedScope:  "global",
				ExpectedBlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
		}

		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})

	t.Run("Get_DeduplicatesActiveKeyLinksWhenPreviousKeyMatches", func(t *testing.T) {
		desName := "test-des"
		des := createAzureDiskEncryptionSetWithPreviousKeysSameVault(desName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, desName, nil).Return(
			armcompute.DiskEncryptionSetsClientGetResponse{DiskEncryptionSet: *des},
			nil,
		)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], desName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		var keyCount, dnsCount int
		for _, liq := range sdpItem.GetLinkedItemQueries() {
			q := liq.GetQuery()
			if q == nil {
				continue
			}

			if q.GetType() == azureshared.KeyVaultKey.String() &&
				q.GetMethod() == sdp.QueryMethod_GET &&
				q.GetQuery() == shared.CompositeLookupKey("test-vault", "test-key") {
				keyCount++
			}

			if q.GetType() == "dns" &&
				q.GetMethod() == sdp.QueryMethod_SEARCH &&
				q.GetQuery() == "test-vault.vault.azure.net" {
				dnsCount++
			}
		}

		if keyCount != 1 {
			t.Fatalf("Expected exactly 1 KeyVaultKey link for ActiveKey/PreviousKeys overlap, got %d", keyCount)
		}
		if dnsCount != 1 {
			t.Fatalf("Expected exactly 1 dns link for ActiveKey/PreviousKeys overlap, got %d", dnsCount)
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting disk encryption set with empty name, but got nil")
		}
	})

	t.Run("WrapperGet_MissingQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Expected no panic, but got: %v", r)
			}
		}()

		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when queryParts is empty, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		desName := "test-des"
		des := &armcompute.DiskEncryptionSet{
			Name:     nil,
			Location: to.Ptr("eastus"),
		}

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, desName, nil).Return(
			armcompute.DiskEncryptionSetsClientGetResponse{DiskEncryptionSet: *des},
			nil,
		)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], desName, true)
		if qErr == nil {
			t.Error("Expected error when disk encryption set has no name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		des1 := createAzureDiskEncryptionSet("test-des-1")
		des2 := createAzureDiskEncryptionSet("test-des-2")

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockPager := newMockDiskEncryptionSetsPager(ctrl, []*armcompute.DiskEncryptionSet{des1, des2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
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
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		des1 := createAzureDiskEncryptionSet("test-des-1")
		desNil := &armcompute.DiskEncryptionSet{
			Name:     nil, // Should be skipped
			Location: to.Ptr("eastus"),
		}

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockPager := newMockDiskEncryptionSetsPager(ctrl, []*armcompute.DiskEncryptionSet{des1, desNil})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}
	})

	t.Run("List_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockPager := newErrorDiskEncryptionSetsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "pager error") {
			t.Fatalf("Expected error to contain 'pager error', got: %v", err)
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		des1 := createAzureDiskEncryptionSet("test-des-1")
		des2 := createAzureDiskEncryptionSet("test-des-2")

		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockPager := newMockDiskEncryptionSetsPager(ctrl, []*armcompute.DiskEncryptionSet{des1, des2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		stream := discovery.NewQueryResultStream(
			func(item *sdp.Item) {
				items = append(items, item)
				wg.Done()
			},
			func(err error) {},
		)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("ListStream_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockDiskEncryptionSetsClient(ctrl)
		mockPager := newErrorDiskEncryptionSetsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskEncryptionSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		errCh := make(chan error, 1)
		stream := discovery.NewQueryResultStream(
			func(item *sdp.Item) {},
			func(err error) { errCh <- err },
		)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)

		select {
		case err := <-errCh:
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), "pager error") {
				t.Fatalf("Expected error to contain 'pager error', got: %v", err)
			}
		default:
			t.Fatalf("Expected an error to be sent to the stream, got none")
		}
	})
}

func createAzureDiskEncryptionSet(name string) *armcompute.DiskEncryptionSet {
	return &armcompute.DiskEncryptionSet{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.EncryptionSetProperties{
			ProvisioningState: to.Ptr("Succeeded"),
		},
	}
}

func createAzureDiskEncryptionSetWithAllLinks(name, subscriptionID, resourceGroup string) *armcompute.DiskEncryptionSet {
	return &armcompute.DiskEncryptionSet{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.EncryptionSetProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			ActiveKey: &armcompute.KeyForDiskEncryptionSet{
				KeyURL: to.Ptr("https://test-vault.vault.azure.net/keys/test-key/00000000000000000000000000000000"),
				SourceVault: &armcompute.SourceVault{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-vault"),
				},
			},
		},
		Identity: &armcompute.EncryptionSetIdentity{
			UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
				"/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
			},
		},
	}
}

func createAzureDiskEncryptionSetWithPreviousKeys(name, subscriptionID, resourceGroup string) *armcompute.DiskEncryptionSet {
	des := createAzureDiskEncryptionSetWithAllLinks(name, subscriptionID, resourceGroup)
	des.Properties.PreviousKeys = []*armcompute.KeyForDiskEncryptionSet{
		{
			KeyURL: to.Ptr("https://test-old-vault.vault.azure.net/keys/test-old-key/00000000000000000000000000000000"),
			SourceVault: &armcompute.SourceVault{
				ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-old-vault"),
			},
		},
	}
	return des
}

func createAzureDiskEncryptionSetWithPreviousKeysSameVault(name, subscriptionID, resourceGroup string) *armcompute.DiskEncryptionSet {
	des := createAzureDiskEncryptionSetWithAllLinks(name, subscriptionID, resourceGroup)
	des.Properties.PreviousKeys = []*armcompute.KeyForDiskEncryptionSet{
		{
			// Same vault + key as ActiveKey.KeyURL to ensure links are deduplicated.
			KeyURL: to.Ptr("https://test-vault.vault.azure.net/keys/test-key/00000000000000000000000000000000"),
			SourceVault: &armcompute.SourceVault{
				ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-vault"),
			},
		},
	}
	return des
}

type mockDiskEncryptionSetsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.DiskEncryptionSet
	index int
	more  bool
}

func newMockDiskEncryptionSetsPager(ctrl *gomock.Controller, items []*armcompute.DiskEncryptionSet) clients.DiskEncryptionSetsPager {
	return &mockDiskEncryptionSetsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockDiskEncryptionSetsPager) More() bool {
	return m.more
}

func (m *mockDiskEncryptionSetsPager) NextPage(ctx context.Context) (armcompute.DiskEncryptionSetsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.DiskEncryptionSetsClientListByResourceGroupResponse{
			DiskEncryptionSetList: armcompute.DiskEncryptionSetList{Value: []*armcompute.DiskEncryptionSet{}},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.DiskEncryptionSetsClientListByResourceGroupResponse{
		DiskEncryptionSetList: armcompute.DiskEncryptionSetList{Value: []*armcompute.DiskEncryptionSet{item}},
	}, nil
}

type errorDiskEncryptionSetsPager struct {
	ctrl *gomock.Controller
}

func newErrorDiskEncryptionSetsPager(ctrl *gomock.Controller) clients.DiskEncryptionSetsPager {
	return &errorDiskEncryptionSetsPager{ctrl: ctrl}
}

func (e *errorDiskEncryptionSetsPager) More() bool { return true }

func (e *errorDiskEncryptionSetsPager) NextPage(ctx context.Context) (armcompute.DiskEncryptionSetsClientListByResourceGroupResponse, error) {
	return armcompute.DiskEncryptionSetsClientListByResourceGroupResponse{}, errors.New("pager error")
}
