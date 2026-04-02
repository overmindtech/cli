package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var DBforPostgreSQLFlexibleServerAdministratorLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServerAdministrator)

type dbforPostgreSQLFlexibleServerAdministratorWrapper struct {
	client clients.DBforPostgreSQLFlexibleServerAdministratorClient

	*azureshared.MultiResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServerAdministrator(client clients.DBforPostgreSQLFlexibleServerAdministratorClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &dbforPostgreSQLFlexibleServerAdministratorWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServerAdministrator,
		),
	}
}

// Get retrieves a single administrator by server name and object ID
// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/administrators-microsoft-entra/get
func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and objectId",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	objectID := queryParts[1]

	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if objectID == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "objectId cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, objectID)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureAdministratorToSDPItem(&resp.AdministratorMicrosoftEntra, serverName, scope)
}

// Search retrieves all administrators for a given server
// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/administrators-microsoft-entra/list-by-server
func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName cannot be empty"), scope, s.Type())
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, admin := range page.Value {
			if admin.Name == nil {
				continue
			}

			item, sdpErr := s.azureAdministratorToSDPItem(admin, serverName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, s.Type()))
		return
	}
	serverName := queryParts[0]
	if serverName == "" {
		stream.SendError(azureshared.QueryError(errors.New("serverName cannot be empty"), scope, s.Type()))
		return
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, admin := range page.Value {
			if admin.Name == nil {
				continue
			}
			item, sdpErr := s.azureAdministratorToSDPItem(admin, serverName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLFlexibleServerAdministratorLookupByName,
	}
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) azureAdministratorToSDPItem(admin *armpostgresqlflexibleservers.AdministratorMicrosoftEntra, serverName, scope string) (*sdp.Item, *sdp.QueryError) {
	if admin.Name == nil {
		return nil, azureshared.QueryError(errors.New("administrator name (objectId) is nil"), scope, s.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(admin)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	objectID := *admin.Name

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, objectID))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            s.Type(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to the parent PostgreSQL Flexible Server
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  serverName,
			Scope:  scope,
		},
	})

	return sdpItem, nil
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.DBforPostgreSQLFlexibleServer,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/databases#microsoftdbforpostgresql
func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/administrators/read",
	}
}

func (s dbforPostgreSQLFlexibleServerAdministratorWrapper) PredefinedRole() string {
	return "Reader"
}
