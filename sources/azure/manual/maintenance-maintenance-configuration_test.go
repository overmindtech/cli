package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance"
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

func TestMaintenanceMaintenanceConfiguration(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		configName := "test-maintenance-config"
		config := createMaintenanceConfiguration(configName)

		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, configName, nil).Return(
			armmaintenance.ConfigurationsClientGetResponse{
				Configuration: *config,
			}, nil)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], configName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.MaintenanceMaintenanceConfiguration.String() {
			t.Errorf("Expected type %s, got %s", azureshared.MaintenanceMaintenanceConfiguration, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != configName {
			t.Errorf("Expected unique attribute value %s, got %s", configName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		config1 := createMaintenanceConfiguration("test-config-1")
		config2 := createMaintenanceConfiguration("test-config-2")

		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)
		mockPager := newMockMaintenanceConfigurationPager(ctrl, []*armmaintenance.Configuration{config1, config2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		config1 := createMaintenanceConfiguration("test-config-1")
		config2 := createMaintenanceConfiguration("test-config-2")

		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)
		mockPager := newMockMaintenanceConfigurationPager(ctrl, []*armmaintenance.Configuration{config1, config2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("resource not found")

		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armmaintenance.ConfigurationsClientGetResponse{}, expectedErr)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting resource with empty name, but got nil")
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		config1 := createMaintenanceConfiguration("test-config-1")
		configNilName := &armmaintenance.Configuration{
			ID:       new(string),
			Name:     nil,
			Location: new(string),
		}

		mockClient := mocks.NewMockMaintenanceConfigurationClient(ctrl)
		mockPager := newMockMaintenanceConfigurationPager(ctrl, []*armmaintenance.Configuration{config1, configNilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewMaintenanceMaintenanceConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}
	})
}

func createMaintenanceConfiguration(name string) *armmaintenance.Configuration {
	location := "eastus"
	maintenanceScope := armmaintenance.MaintenanceScopeHost
	visibility := armmaintenance.VisibilityCustom
	configID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Maintenance/maintenanceConfigurations/" + name

	return &armmaintenance.Configuration{
		ID:       &configID,
		Name:     &name,
		Location: &location,
		Type:     new("Microsoft.Maintenance/maintenanceConfigurations"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armmaintenance.ConfigurationProperties{
			MaintenanceScope: &maintenanceScope,
			Visibility:       &visibility,
			Namespace:        new("Microsoft.Compute"),
			MaintenanceWindow: &armmaintenance.Window{
				StartDateTime: new("2025-01-01 00:00"),
				Duration:      new("02:00"),
				TimeZone:      new("Pacific Standard Time"),
				RecurEvery:    new("Day"),
			},
		},
	}
}

type mockMaintenanceConfigurationPager struct {
	ctrl  *gomock.Controller
	items []*armmaintenance.Configuration
	index int
	more  bool
}

func newMockMaintenanceConfigurationPager(ctrl *gomock.Controller, items []*armmaintenance.Configuration) clients.MaintenanceConfigurationPager {
	return &mockMaintenanceConfigurationPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockMaintenanceConfigurationPager) More() bool {
	return m.more
}

func (m *mockMaintenanceConfigurationPager) NextPage(ctx context.Context) (armmaintenance.ConfigurationsForResourceGroupClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armmaintenance.ConfigurationsForResourceGroupClientListResponse{
			ListMaintenanceConfigurationsResult: armmaintenance.ListMaintenanceConfigurationsResult{
				Value: []*armmaintenance.Configuration{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armmaintenance.ConfigurationsForResourceGroupClientListResponse{
		ListMaintenanceConfigurationsResult: armmaintenance.ListMaintenanceConfigurationsResult{
			Value: []*armmaintenance.Configuration{item},
		},
	}, nil
}
