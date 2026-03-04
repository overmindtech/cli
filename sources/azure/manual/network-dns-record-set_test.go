package manual_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func createAzureRecordSet(relativeName, recordType, zoneName, subscriptionID, resourceGroup string) *armdns.RecordSet {
	fqdn := relativeName + "." + zoneName
	armType := "Microsoft.Network/dnszones/" + recordType
	provisioningState := "Succeeded"
	return &armdns.RecordSet{
		ID:   new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/dnszones/" + zoneName + "/" + recordType + "/" + relativeName),
		Name: new(relativeName),
		Type: new(armType),
		Properties: &armdns.RecordSetProperties{
			Fqdn:              new(fqdn),
			ProvisioningState: &provisioningState,
			TTL:               new(int64(3600)),
			ARecords:          nil,
			AaaaRecords:       nil,
			CnameRecord:       nil,
			MxRecords:         nil,
			NsRecords:         nil,
			PtrRecords:        nil,
			SoaRecord:         nil,
			SrvRecords:        nil,
			TxtRecords:        nil,
			CaaRecords:        nil,
			TargetResource:    nil,
			Metadata:          nil,
		},
	}
}

type mockRecordSetsPager struct {
	pages []armdns.RecordSetsClientListAllByDNSZoneResponse
	index int
}

func (m *mockRecordSetsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockRecordSetsPager) NextPage(ctx context.Context) (armdns.RecordSetsClientListAllByDNSZoneResponse, error) {
	if m.index >= len(m.pages) {
		return armdns.RecordSetsClientListAllByDNSZoneResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func TestNetworkDNSRecordSet(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	zoneName := "example.com"
	relativeName := "www"
	recordType := "A"
	query := shared.CompositeLookupKey(zoneName, recordType, relativeName)

	t.Run("Get", func(t *testing.T) {
		rs := createAzureRecordSet(relativeName, recordType, zoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, relativeName, armdns.RecordType(recordType), nil).Return(
			armdns.RecordSetsClientGetResponse{
				RecordSet: *rs,
			}, nil)

		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkDNSRecordSet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDNSRecordSet.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(zoneName, recordType, relativeName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkZone.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  zoneName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "www.example.com",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Single part (zone only) is insufficient
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], zoneName, true)
		if qErr == nil {
			t.Error("Expected error when providing only one query part, got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rs1 := createAzureRecordSet("www", "A", zoneName, subscriptionID, resourceGroup)
		rs2 := createAzureRecordSet("mail", "MX", zoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		mockPager := &mockRecordSetsPager{
			pages: []armdns.RecordSetsClientListAllByDNSZoneResponse{
				{
					RecordSetListResult: armdns.RecordSetListResult{
						Value: []*armdns.RecordSet{rs1, rs2},
					},
				},
			},
		}
		mockClient.EXPECT().NewListAllByDNSZonePager(resourceGroup, zoneName, nil).Return(mockPager)

		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not implement SearchableAdapter")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], zoneName, true)
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
		rs := createAzureRecordSet("www", "A", zoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		mockPager := &mockRecordSetsPager{
			pages: []armdns.RecordSetsClientListAllByDNSZoneResponse{
				{
					RecordSetListResult: armdns.RecordSetListResult{
						Value: []*armdns.RecordSet{rs},
					},
				},
			},
		}
		mockClient.EXPECT().NewListAllByDNSZonePager(resourceGroup, zoneName, nil).Return(mockPager)

		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		streamable.SearchStream(ctx, wrapper.Scopes()[0], zoneName, true, stream)

		if len(received) != 1 {
			t.Fatalf("Expected 1 item from SearchStream, got %d", len(received))
		}
		if received[0].GetType() != azureshared.NetworkDNSRecordSet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDNSRecordSet.String(), received[0].GetType())
		}
	})

	t.Run("SearchWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, qErr := searchable.Search(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when providing empty zone name, got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("record set not found")
		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, relativeName, armdns.RecordType(recordType), nil).Return(
			armdns.RecordSetsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when Get fails, got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		potentialLinks := wrapper.PotentialLinks()
		if !potentialLinks[azureshared.NetworkZone] {
			t.Error("Expected PotentialLinks to include NetworkZone")
		}
		if !potentialLinks[stdlib.NetworkDNS] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkDNS")
		}
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkIP")
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		perms := wrapper.IAMPermissions()
		if len(perms) == 0 {
			t.Error("Expected at least one IAM permission")
		}
		expectedPermission := "Microsoft.Network/dnszones/*/read"
		found := slices.Contains(perms, expectedPermission)
		if !found {
			t.Errorf("Expected IAMPermissions to include %q", expectedPermission)
		}
	})

	t.Run("GetWithARecordsAndCnameLinkedQueries", func(t *testing.T) {
		rs := createAzureRecordSet(relativeName, recordType, zoneName, subscriptionID, resourceGroup)
		rs.Properties.ARecords = []*armdns.ARecord{{IPv4Address: new("192.168.1.1")}}
		rs.Properties.CnameRecord = &armdns.CnameRecord{Cname: new("backend.example.com")}

		mockClient := mocks.NewMockRecordSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, relativeName, armdns.RecordType(recordType), nil).Return(
			armdns.RecordSetsClientGetResponse{RecordSet: *rs}, nil)

		wrapper := manual.NewNetworkDNSRecordSet(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		var hasIPLink, hasCnameLink bool
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			q := lq.GetQuery()
			if q == nil {
				continue
			}
			if q.GetType() == stdlib.NetworkIP.String() && q.GetQuery() == "192.168.1.1" && q.GetMethod() == sdp.QueryMethod_GET && q.GetScope() == "global" {
				hasIPLink = true
			}
			if q.GetType() == stdlib.NetworkDNS.String() && q.GetQuery() == "backend.example.com" && q.GetMethod() == sdp.QueryMethod_SEARCH && q.GetScope() == "global" {
				hasCnameLink = true
			}
		}
		if !hasIPLink {
			t.Error("Expected LinkedItemQueries to include stdlib.NetworkIP for A record 192.168.1.1 (GET, global)")
		}
		if !hasCnameLink {
			t.Error("Expected LinkedItemQueries to include stdlib.NetworkDNS for CNAME backend.example.com (SEARCH, global)")
		}
	})
}
