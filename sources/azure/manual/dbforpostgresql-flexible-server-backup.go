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

var DBforPostgreSQLFlexibleServerBackupLookupByName = shared.NewItemTypeLookup("name", azureshared.DBforPostgreSQLFlexibleServerBackup)

type dbforPostgreSQLFlexibleServerBackupWrapper struct {
	client clients.DBforPostgreSQLFlexibleServerBackupClient

	*azureshared.MultiResourceGroupBase
}

func NewDBforPostgreSQLFlexibleServerBackup(client clients.DBforPostgreSQLFlexibleServerBackupClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &dbforPostgreSQLFlexibleServerBackupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DBforPostgreSQLFlexibleServerBackup,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/backups-automatic-and-on-demand/get?view=rest-postgresql-2025-08-01
func (s dbforPostgreSQLFlexibleServerBackupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and backupName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	backupName := queryParts[1]
	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	if backupName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "backupName cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, backupName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureBackupToSDPItem(&resp.BackupAutomaticAndOnDemand, serverName, backupName, scope)
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) azureBackupToSDPItem(backup *armpostgresqlflexibleservers.BackupAutomaticAndOnDemand, serverName, backupName, scope string) (*sdp.Item, *sdp.QueryError) {
	if backup.Name == nil {
		return nil, azureshared.QueryError(errors.New("backup name is nil"), scope, s.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(backup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, backupName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DBforPostgreSQLFlexibleServerBackup.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            nil,
	}

	// Link to parent PostgreSQL Flexible Server
	if backup.ID != nil {
		params := azureshared.ExtractPathParamsFromResourceID(*backup.ID, []string{"flexibleServers"})
		if len(params) > 0 {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  params[0],
					Scope:  scope,
				},
			})
		}
	} else {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.DBforPostgreSQLFlexibleServer.String(),
				Method: sdp.QueryMethod_GET,
				Query:  serverName,
				Scope:  scope,
			},
		})
	}

	return sdpItem, nil
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DBforPostgreSQLFlexibleServerLookupByName,
		DBforPostgreSQLFlexibleServerBackupLookupByName,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/postgresql/backups-automatic-and-on-demand/list-by-server?view=rest-postgresql-2025-08-01
func (s dbforPostgreSQLFlexibleServerBackupWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]

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
		for _, backup := range page.Value {
			if backup.Name == nil {
				continue
			}
			item, sdpErr := s.azureBackupToSDPItem(backup, serverName, *backup.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, s.Type()))
		return
	}
	serverName := queryParts[0]

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
		for _, backup := range page.Value {
			if backup.Name == nil {
				continue
			}
			item, sdpErr := s.azureBackupToSDPItem(backup, serverName, *backup.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			DBforPostgreSQLFlexibleServerLookupByName,
		},
	}
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.DBforPostgreSQLFlexibleServer: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftdbforpostgresql
func (s dbforPostgreSQLFlexibleServerBackupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.DBforPostgreSQL/flexibleServers/backups/read",
	}
}

func (s dbforPostgreSQLFlexibleServerBackupWrapper) PredefinedRole() string {
	return "Reader"
}
