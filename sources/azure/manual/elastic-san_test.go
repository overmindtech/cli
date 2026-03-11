package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
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

func createAzureElasticSan(name string) *armelasticsan.ElasticSan {
	baseSize := int64(1)
	extendedSize := int64(2)
	provisioningState := armelasticsan.ProvisioningStatesSucceeded
	return &armelasticsan.ElasticSan{
		ID:       new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ElasticSan/elasticSans/" + name),
		Name:     new(name),
		Location: new("eastus"),
		Type:     new("Microsoft.ElasticSan/elasticSans"),
		Tags:     map[string]*string{"env": new("test")},
		Properties: &armelasticsan.Properties{
			BaseSizeTiB:             &baseSize,
			ExtendedCapacitySizeTiB:  &extendedSize,
			ProvisioningState:        &provisioningState,
			VolumeGroupCount:         new(int64(0)),
		},
	}
}

func createAzureElasticSanWithPrivateEndpoint(name, subscriptionID, resourceGroup string) *armelasticsan.ElasticSan {
	es := createAzureElasticSan(name)
	es.Properties.PrivateEndpointConnections = []*armelasticsan.PrivateEndpointConnection{
		{
			ID:   new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ElasticSan/elasticSans/" + name + "/privateEndpointConnections/pec-1"),
			Name: new("pec-1"),
			Properties: &armelasticsan.PrivateEndpointConnectionProperties{
				PrivateEndpoint: &armelasticsan.PrivateEndpoint{
					ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"),
				},
			},
		},
	}
	return es
}

type mockElasticSanPager struct {
	pages []armelasticsan.ElasticSansClientListByResourceGroupResponse
	index int
}

func (m *mockElasticSanPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockElasticSanPager) NextPage(ctx context.Context) (armelasticsan.ElasticSansClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armelasticsan.ElasticSansClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func TestElasticSan(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		elasticSanName := "test-elastic-san"
		es := createAzureElasticSan(elasticSanName)

		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, nil).Return(
			armelasticsan.ElasticSansClientGetResponse{
				ElasticSan: *es,
			}, nil)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, elasticSanName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ElasticSan.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSan.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != elasticSanName {
			t.Errorf("Expected unique attribute value %s, got %s", elasticSanName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			// ElasticSanVolumeGroup SEARCH link (parent→child); no private endpoints in createAzureElasticSan
			shared.RunStaticTests(t, adapter, sdpItem, shared.QueryTests{
				{
					ExpectedType:   azureshared.ElasticSanVolumeGroup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  elasticSanName,
					ExpectedScope:  scope,
				},
			})
		})
	})

	t.Run("GetWithPrivateEndpointLink", func(t *testing.T) {
		elasticSanName := "test-elastic-san-pe"
		es := createAzureElasticSanWithPrivateEndpoint(elasticSanName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, nil).Return(
			armelasticsan.ElasticSansClientGetResponse{
				ElasticSan: *es,
			}, nil)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, elasticSanName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{
				ExpectedType:   azureshared.ElasticSanVolumeGroup.String(),
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  elasticSanName,
				ExpectedScope:  scope,
			},
			{
				ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "test-pe",
				ExpectedScope:  scope,
			},
		}
		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})

	t.Run("List", func(t *testing.T) {
		es1 := createAzureElasticSan("es-1")
		es2 := createAzureElasticSan("es-2")

		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockPager := &mockElasticSanPager{
			pages: []armelasticsan.ElasticSansClientListByResourceGroupResponse{
				{List: armelasticsan.List{Value: []*armelasticsan.ElasticSan{es1, es2}}},
			},
		}
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		es := createAzureElasticSan("es-stream")

		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockPager := &mockElasticSanPager{
			pages: []armelasticsan.ElasticSansClientListByResourceGroupResponse{
				{List: armelasticsan.List{Value: []*armelasticsan.ElasticSan{es}}},
			},
		}
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(1)

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

		listStreamable.ListStream(ctx, scope, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(items))
		}

		if items[0].GetType() != azureshared.ElasticSan.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSan.String(), items[0].GetType())
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		es1 := createAzureElasticSan("es-1")
		esNilName := &armelasticsan.ElasticSan{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
		}

		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockPager := &mockElasticSanPager{
			pages: []armelasticsan.ElasticSansClientListByResourceGroupResponse{
				{List: armelasticsan.List{Value: []*armelasticsan.ElasticSan{es1, esNilName}}},
			},
		}
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armelasticsan.ElasticSansClientGetResponse{}, errors.New("elastic san not found"))

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent Elastic SAN, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanClient(ctrl)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting Elastic SAN with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanClient(ctrl)

		wrapper := manual.NewElasticSan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when getting Elastic SAN with insufficient query parts, but got nil")
		}
	})
}
