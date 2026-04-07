package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var MaintenanceMaintenanceConfigurationLookupByName = shared.NewItemTypeLookup("name", azureshared.MaintenanceMaintenanceConfiguration)

type maintenanceMaintenanceConfigurationWrapper struct {
	client clients.MaintenanceConfigurationClient

	*azureshared.MultiResourceGroupBase
}

func NewMaintenanceMaintenanceConfiguration(client clients.MaintenanceConfigurationClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &maintenanceMaintenanceConfigurationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			azureshared.MaintenanceMaintenanceConfiguration,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/maintenance/maintenance-configurations-for-resource-group/list
func (c maintenanceMaintenanceConfigurationWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, config := range page.Value {
			if config.Name == nil {
				continue
			}
			item, sdpErr := c.azureMaintenanceConfigurationToSDPItem(config, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c maintenanceMaintenanceConfigurationWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, config := range page.Value {
			if config.Name == nil {
				continue
			}
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureMaintenanceConfigurationToSDPItem(config, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/maintenance/maintenance-configurations/get
func (c maintenanceMaintenanceConfigurationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the maintenance configuration name"), scope, c.Type())
	}
	configName := queryParts[0]
	if configName == "" {
		return nil, azureshared.QueryError(errors.New("configName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	result, err := c.client.Get(ctx, rgScope.ResourceGroup, configName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureMaintenanceConfigurationToSDPItem(&result.Configuration, scope)
}

func (c maintenanceMaintenanceConfigurationWrapper) azureMaintenanceConfigurationToSDPItem(config *armmaintenance.Configuration, scope string) (*sdp.Item, *sdp.QueryError) {
	if config.Name == nil {
		return nil, azureshared.QueryError(errors.New("maintenance configuration name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(config, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.MaintenanceMaintenanceConfiguration.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(config.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	return sdpItem, nil
}

func (c maintenanceMaintenanceConfigurationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		MaintenanceMaintenanceConfigurationLookupByName,
	}
}

func (c maintenanceMaintenanceConfigurationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet()
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftmaintenance
func (c maintenanceMaintenanceConfigurationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Maintenance/maintenanceConfigurations/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles
func (c maintenanceMaintenanceConfigurationWrapper) PredefinedRole() string {
	return "Reader"
}
