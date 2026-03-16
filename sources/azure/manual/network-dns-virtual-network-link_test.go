package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
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

func createAzureVirtualNetworkLink(name, privateZoneName, subscriptionID, resourceGroup string) *armprivatedns.VirtualNetworkLink {
	provisioningState := armprivatedns.ProvisioningStateSucceeded
	linkState := armprivatedns.VirtualNetworkLinkStateCompleted
	registrationEnabled := true
	return &armprivatedns.VirtualNetworkLink{
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateDnsZones/" + privateZoneName + "/virtualNetworkLinks/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/privateDnsZones/virtualNetworkLinks"),
		Location: new("global"),
		Tags:     map[string]*string{"env": new("test")},
		Properties: &armprivatedns.VirtualNetworkLinkProperties{
			ProvisioningState:   &provisioningState,
			VirtualNetworkLinkState: &linkState,
			RegistrationEnabled: &registrationEnabled,
			VirtualNetwork: &armprivatedns.SubResource{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet"),
			},
		},
	}
}

type mockVirtualNetworkLinksPager struct {
	pages []armprivatedns.VirtualNetworkLinksClientListResponse
	index int
}

func (m *mockVirtualNetworkLinksPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockVirtualNetworkLinksPager) NextPage(_ context.Context) (armprivatedns.VirtualNetworkLinksClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armprivatedns.VirtualNetworkLinksClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func newMockVirtualNetworkLinksPager(_ *gomock.Controller, items []*armprivatedns.VirtualNetworkLink) clients.VirtualNetworkLinksPager {
	return &mockVirtualNetworkLinksPager{
		pages: []armprivatedns.VirtualNetworkLinksClientListResponse{
			{
				VirtualNetworkLinkListResult: armprivatedns.VirtualNetworkLinkListResult{
					Value: items,
				},
			},
		},
	}
}

func TestNetworkDNSVirtualNetworkLink(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	privateZoneName := "example.private.zone"
	linkName := "test-link"
	query := shared.CompositeLookupKey(privateZoneName, linkName)

	t.Run("Get", func(t *testing.T) {
		link := createAzureVirtualNetworkLink(linkName, privateZoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, privateZoneName, linkName, nil).Return(
			armprivatedns.VirtualNetworkLinksClientGetResponse{
				VirtualNetworkLink: *link,
			}, nil)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkDNSVirtualNetworkLink.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDNSVirtualNetworkLink.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(privateZoneName, linkName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkPrivateDNSZone.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  privateZoneName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], privateZoneName, true)
		if qErr == nil {
			t.Error("Expected error when providing only one query part, got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		emptyQuery := shared.CompositeLookupKey(privateZoneName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], emptyQuery, true)
		if qErr == nil {
			t.Error("Expected error when getting resource with empty link name, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		link1 := createAzureVirtualNetworkLink("link-1", privateZoneName, subscriptionID, resourceGroup)
		link2 := createAzureVirtualNetworkLink("link-2", privateZoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockPager := newMockVirtualNetworkLinksPager(ctrl, []*armprivatedns.VirtualNetworkLink{link1, link2})
		mockClient.EXPECT().NewListPager(resourceGroup, privateZoneName, nil).Return(mockPager)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not implement SearchableAdapter")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], privateZoneName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected valid item, got: %v", item.Validate())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		link := createAzureVirtualNetworkLink(linkName, privateZoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockPager := newMockVirtualNetworkLinksPager(ctrl, []*armprivatedns.VirtualNetworkLink{link})
		mockClient.EXPECT().NewListPager(resourceGroup, privateZoneName, nil).Return(mockPager)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		streamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not implement SearchStreamableAdapter")
		}

		var received []*sdp.Item
		stream := discovery.NewQueryResultStream(
			func(item *sdp.Item) { received = append(received, item) },
			func(error) {},
		)
		streamable.SearchStream(ctx, wrapper.Scopes()[0], privateZoneName, true, stream)

		if len(received) != 1 {
			t.Fatalf("Expected 1 item from SearchStream, got %d", len(received))
		}
		if received[0].GetType() != azureshared.NetworkDNSVirtualNetworkLink.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDNSVirtualNetworkLink.String(), received[0].GetType())
		}
	})

	t.Run("SearchWithEmptyZoneName", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when providing empty zone name, got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("virtual network link not found")
		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, privateZoneName, linkName, nil).Return(
			armprivatedns.VirtualNetworkLinksClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when Get fails, got nil")
		}
	})

	t.Run("GetWithCrossResourceGroupVNet", func(t *testing.T) {
		link := createAzureVirtualNetworkLink(linkName, privateZoneName, subscriptionID, resourceGroup)
		link.Properties.VirtualNetwork = &armprivatedns.SubResource{
			ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Network/virtualNetworks/cross-rg-vnet"),
		}

		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, privateZoneName, linkName, nil).Return(
			armprivatedns.VirtualNetworkLinksClientGetResponse{
				VirtualNetworkLink: *link,
			}, nil)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		var hasVNetLink bool
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			q := lq.GetQuery()
			if q.GetType() == azureshared.NetworkVirtualNetwork.String() && q.GetQuery() == "cross-rg-vnet" && q.GetScope() == subscriptionID+".other-rg" {
				hasVNetLink = true
			}
		}
		if !hasVNetLink {
			t.Error("Expected LinkedItemQueries to include VirtualNetwork with cross-resource-group scope")
		}
	})

	t.Run("GetWithoutVirtualNetwork", func(t *testing.T) {
		link := createAzureVirtualNetworkLink(linkName, privateZoneName, subscriptionID, resourceGroup)
		link.Properties.VirtualNetwork = nil

		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, privateZoneName, linkName, nil).Return(
			armprivatedns.VirtualNetworkLinksClientGetResponse{
				VirtualNetworkLink: *link,
			}, nil)

		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have only the parent DNS zone link, not a VNet link
		vnetLinks := 0
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			if lq.GetQuery().GetType() == azureshared.NetworkVirtualNetwork.String() {
				vnetLinks++
			}
		}
		if vnetLinks != 0 {
			t.Errorf("Expected no VirtualNetwork linked queries when VirtualNetwork is nil, got %d", vnetLinks)
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkLinksClient(ctrl)
		wrapper := manual.NewNetworkDNSVirtualNetworkLink(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		potentialLinks := wrapper.PotentialLinks()
		if !potentialLinks[azureshared.NetworkPrivateDNSZone] {
			t.Error("Expected PotentialLinks to include NetworkPrivateDNSZone")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetwork] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetwork")
		}
	})
}
