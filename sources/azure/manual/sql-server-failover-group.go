package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var SQLServerFailoverGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServerFailoverGroup)

type sqlServerFailoverGroupWrapper struct {
	client clients.SqlFailoverGroupsClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlServerFailoverGroup(client clients.SqlFailoverGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlServerFailoverGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServerFailoverGroup,
		),
	}
}

// Get retrieves a specific failover group by server name and failover group name
// ref: https://learn.microsoft.com/en-us/rest/api/sql/failover-groups/get
func (c sqlServerFailoverGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and failoverGroupName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	failoverGroupName := queryParts[1]
	if failoverGroupName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "failoverGroupName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, serverName, failoverGroupName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureFailoverGroupToSDPItem(&resp.FailoverGroup, serverName, scope)
}

// Search retrieves all failover groups for a given server
// ref: https://learn.microsoft.com/en-us/rest/api/sql/failover-groups/list-by-server
func (c sqlServerFailoverGroupWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, failoverGroup := range page.Value {
			if failoverGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureFailoverGroupToSDPItem(failoverGroup, serverName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// SearchStream streams all failover groups for a given server
func (c sqlServerFailoverGroupWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: serverName"), scope, c.Type()))
		return
	}
	serverName := queryParts[0]
	if serverName == "" {
		stream.SendError(azureshared.QueryError(errors.New("serverName cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.ListByServer(ctx, rgScope.ResourceGroup, serverName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, failoverGroup := range page.Value {
			if failoverGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureFailoverGroupToSDPItem(failoverGroup, serverName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c sqlServerFailoverGroupWrapper) azureFailoverGroupToSDPItem(failoverGroup *armsql.FailoverGroup, serverName, scope string) (*sdp.Item, *sdp.QueryError) {
	if failoverGroup.Name == nil {
		return nil, azureshared.QueryError(errors.New("failover group name is nil"), scope, c.Type())
	}
	failoverGroupName := *failoverGroup.Name

	attributes, err := shared.ToAttributesWithExclude(failoverGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, failoverGroupName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServerFailoverGroup.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(failoverGroup.Tags),
	}

	// Health mapping based on replication state
	if failoverGroup.Properties != nil && failoverGroup.Properties.ReplicationState != nil {
		switch *failoverGroup.Properties.ReplicationState {
		case "CATCH_UP", "PENDING", "SEEDING":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "SUSPENDED":
			sdpItem.Health = sdp.Health_HEALTH_WARNING.Enum()
		case "":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link back to the parent SQL Server
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.SQLServer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  serverName,
			Scope:  scope,
		},
	})

	if failoverGroup.Properties != nil {
		// Link to partner servers
		if failoverGroup.Properties.PartnerServers != nil {
			for _, partner := range failoverGroup.Properties.PartnerServers {
				if partner != nil && partner.ID != nil && *partner.ID != "" {
					partnerServerName := azureshared.ExtractResourceName(*partner.ID)
					if partnerServerName != "" {
						linkedScope := azureshared.ExtractScopeFromResourceID(*partner.ID)
						if linkedScope == "" {
							linkedScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.SQLServer.String(),
								Method: sdp.QueryMethod_GET,
								Query:  partnerServerName,
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}

		// Link to databases in the failover group
		if failoverGroup.Properties.Databases != nil {
			for _, databaseID := range failoverGroup.Properties.Databases {
				if databaseID != nil && *databaseID != "" {
					// Extract server name and database name from the database resource ID
					params := azureshared.ExtractPathParamsFromResourceID(*databaseID, []string{"servers", "databases"})
					if len(params) >= 2 {
						dbServerName := params[0]
						dbName := params[1]
						linkedScope := azureshared.ExtractScopeFromResourceID(*databaseID)
						if linkedScope == "" {
							linkedScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.SQLDatabase.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(dbServerName, dbName),
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}

		// Link to read-only endpoint target server if specified
		if failoverGroup.Properties.ReadOnlyEndpoint != nil && failoverGroup.Properties.ReadOnlyEndpoint.TargetServer != nil && *failoverGroup.Properties.ReadOnlyEndpoint.TargetServer != "" {
			// TargetServer is a resource ID
			targetServerName := azureshared.ExtractResourceName(*failoverGroup.Properties.ReadOnlyEndpoint.TargetServer)
			if targetServerName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(*failoverGroup.Properties.ReadOnlyEndpoint.TargetServer)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.SQLServer.String(),
						Method: sdp.QueryMethod_GET,
						Query:  targetServerName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (c sqlServerFailoverGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLServerFailoverGroupLookupByName,
	}
}

func (c sqlServerFailoverGroupWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (c sqlServerFailoverGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.SQLServer,
		azureshared.SQLDatabase,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftsql
func (c sqlServerFailoverGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/failoverGroups/read",
	}
}

func (c sqlServerFailoverGroupWrapper) PredefinedRole() string {
	return "Reader"
}
