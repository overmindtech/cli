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

var SQLServerKeyLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServerKey)

type sqlServerKeyWrapper struct {
	client clients.SqlServerKeysClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlServerKey(client clients.SqlServerKeysClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlServerKeyWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServerKey,
		),
	}
}

// Get retrieves a single SQL Server Key by serverName and keyName
// ref: https://learn.microsoft.com/en-us/rest/api/sql/server-keys/get
func (c sqlServerKeyWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and keyName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	serverName := queryParts[0]
	keyName := queryParts[1]

	if serverName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "serverName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	if keyName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "keyName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, serverName, keyName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureSqlServerKeyToSDPItem(&resp.ServerKey, serverName, scope)
}

func (c sqlServerKeyWrapper) azureSqlServerKeyToSDPItem(serverKey *armsql.ServerKey, serverName, scope string) (*sdp.Item, *sdp.QueryError) {
	if serverKey.Name == nil {
		return nil, azureshared.QueryError(errors.New("server key name is nil"), scope, c.Type())
	}
	keyName := *serverKey.Name

	attributes, err := shared.ToAttributesWithExclude(serverKey)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, keyName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServerKey.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link back to parent SQL Server
	if serverName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServer.String(),
				Method: sdp.QueryMethod_GET,
				Query:  serverName,
				Scope:  scope,
			},
		})
	}

	// Link to Key Vault Key if this is an Azure Key Vault type key
	// The URI field contains the Key Vault key URI for AzureKeyVault server key types
	// URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
	if serverKey.Properties != nil && serverKey.Properties.URI != nil && *serverKey.Properties.URI != "" {
		keyURI := *serverKey.Properties.URI
		vaultName := azureshared.ExtractVaultNameFromURI(keyURI)
		keyVaultKeyName := azureshared.ExtractKeyNameFromURI(keyURI)
		if vaultName != "" && keyVaultKeyName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(vaultName, keyVaultKeyName),
					Scope:  scope,
				},
			})
		}
	}

	return sdpItem, nil
}

// Search retrieves all SQL Server Keys for a given server
// ref: https://learn.microsoft.com/en-us/rest/api/sql/server-keys/list-by-server
func (c sqlServerKeyWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
	pager := c.client.NewListByServerPager(rgScope.ResourceGroup, serverName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, serverKey := range page.Value {
			if serverKey.Name == nil {
				continue
			}
			item, sdpErr := c.azureSqlServerKeyToSDPItem(serverKey, serverName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c sqlServerKeyWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
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
	pager := c.client.NewListByServerPager(rgScope.ResourceGroup, serverName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, serverKey := range page.Value {
			if serverKey.Name == nil {
				continue
			}
			item, sdpErr := c.azureSqlServerKeyToSDPItem(serverKey, serverName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c sqlServerKeyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLServerKeyLookupByName,
	}
}

func (c sqlServerKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (c sqlServerKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.SQLServer,
		azureshared.KeyVaultKey,
	)
}

// IAMPermissions returns the required Azure RBAC permissions for reading SQL Server Keys
// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftsql
func (c sqlServerKeyWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/keys/read",
	}
}

func (c sqlServerKeyWrapper) PredefinedRole() string {
	return "Reader"
}
