package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type elasticSanWrapper struct {
	client clients.ElasticSanClient

	*azureshared.MultiResourceGroupBase
}

func NewElasticSan(client clients.ElasticSanClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &elasticSanWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ElasticSan,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/list-by-resource-group
func (e elasticSanWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	pager := e.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, e.Type())
		}
		for _, elasticSan := range page.Value {
			if elasticSan.Name == nil {
				continue
			}
			item, sdpErr := e.azureElasticSanToSDPItem(elasticSan, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (e elasticSanWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, e.Type()))
		return
	}
	pager := e.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, e.Type()))
			return
		}

		for _, elasticSan := range page.Value {
			if elasticSan.Name == nil {
				continue
			}
			item, sdpErr := e.azureElasticSanToSDPItem(elasticSan, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/get
func (e elasticSanWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the Elastic SAN name"), scope, e.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type())
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	resp, err := e.client.Get(ctx, rgScope.ResourceGroup, elasticSanName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	return e.azureElasticSanToSDPItem(&resp.ElasticSan, scope)
}

func (e elasticSanWrapper) azureElasticSanToSDPItem(elasticSan *armelasticsan.ElasticSan, scope string) (*sdp.Item, *sdp.QueryError) {
	if elasticSan.Name == nil {
		return nil, azureshared.QueryError(errors.New("elasticSan name is nil"), scope, e.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(elasticSan, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}

	item := &sdp.Item{
		Type:              azureshared.ElasticSan.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(elasticSan.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to Private Endpoints via PrivateEndpointConnections
	if elasticSan.Properties != nil && elasticSan.Properties.PrivateEndpointConnections != nil {
		for _, pec := range elasticSan.Properties.PrivateEndpointConnections {
			if pec != nil && pec.Properties != nil && pec.Properties.PrivateEndpoint != nil && pec.Properties.PrivateEndpoint.ID != nil {
				peName := azureshared.ExtractResourceName(*pec.Properties.PrivateEndpoint.ID)
				if peName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*pec.Properties.PrivateEndpoint.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  peName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Link to child Volume Groups (SEARCH by parent Elastic SAN name)
	if elasticSan.Name != nil && *elasticSan.Name != "" {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.ElasticSanVolumeGroup.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *elasticSan.Name,
				Scope:  scope,
			},
		})
	}

	// Health from provisioning state
	if elasticSan.Properties != nil && elasticSan.Properties.ProvisioningState != nil {
		switch *elasticSan.Properties.ProvisioningState {
		case armelasticsan.ProvisioningStatesSucceeded:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case armelasticsan.ProvisioningStatesCreating, armelasticsan.ProvisioningStatesUpdating, armelasticsan.ProvisioningStatesDeleting,
			armelasticsan.ProvisioningStatesPending, armelasticsan.ProvisioningStatesRestoring:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armelasticsan.ProvisioningStatesFailed, armelasticsan.ProvisioningStatesCanceled,
			armelasticsan.ProvisioningStatesDeleted, armelasticsan.ProvisioningStatesInvalid:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return item, nil
}

func (e elasticSanWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ElasticSanLookupByName, // defined in elastic-san-volume-snapshot.go
	}
}

func (e elasticSanWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ElasticSanVolumeGroup,
		azureshared.NetworkPrivateEndpoint,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/elastic_san
func (e elasticSanWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_elastic_san.name",
		},
	}
}

func (e elasticSanWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ElasticSan/elasticSans/read",
	}
}

func (e elasticSanWrapper) PredefinedRole() string {
	return "Reader"
}
