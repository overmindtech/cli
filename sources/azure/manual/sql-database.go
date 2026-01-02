package manual

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	SQLServerLookupByName   = shared.NewItemTypeLookup("name", azureshared.SQLServer) //todo: move to sql server adapter when made
	SQLDatabaseLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLDatabase)
)

type sqlDatabaseWrapper struct {
	client clients.SqlDatabasesClient

	*azureshared.ResourceGroupBase
}

func NewSqlDatabase(client clients.SqlDatabasesClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &sqlDatabaseWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLDatabase,
		),
	}
}

func (s sqlDatabaseWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and databaseName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]

	resp, err := s.client.Get(ctx, s.ResourceGroup(), serverName, databaseName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	return s.azureSqlDatabaseToSDPItem(&resp.Database, serverName, databaseName)
}

func (s sqlDatabaseWrapper) azureSqlDatabaseToSDPItem(database *armsql.Database, serverName, databaseName string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(database, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, databaseName))
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLDatabase.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           s.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(database.Tags),
	}

	// Extract server name from database ID
	if database.ID != nil {
		extractedServerName := azureshared.ExtractSQLServerNameFromDatabaseID(*database.ID)
		if extractedServerName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLServer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  extractedServerName,
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // SQL Server changes (especially deletion) affect the database's availability and configuration
					Out: false, // Database changes don't affect the SQL Server itself
				}, // SQL Database is a child resource that depends on its parent SQL Server
			})
		}
	}

	if database.Properties != nil && database.Properties.ElasticPoolID != nil {
		elasticPoolName := azureshared.ExtractSQLElasticPoolNameFromID(*database.Properties.ElasticPoolID)
		if elasticPoolName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLElasticPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  elasticPoolName,
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Elastic pool changes (especially deletion or resource configuration changes) affect the database's performance and availability
					Out: false, // Database changes don't affect the elastic pool itself (though they may affect pool resource usage)
				}, // SQL Database depends on its Elastic Pool for resource allocation and management
			})
		}
	}

	if database.Properties != nil && database.Properties.RecoverableDatabaseID != nil {
		// Extract server name and database name from RecoverableDatabaseID resource ID
		// This handles cross-server scenarios where geo-replicated backups exist on different servers
		// RecoverableDatabaseID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Sql/servers/{serverName}/recoverableDatabases/{databaseName}
		recoverableServerName, recoverableDatabaseName := azureshared.ExtractSQLRecoverableDatabaseInfoFromResourceID(*database.Properties.RecoverableDatabaseID)
		if recoverableServerName != "" && recoverableDatabaseName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLRecoverableDatabase.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(recoverableServerName, recoverableDatabaseName),
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Recoverable database deletion or unavailability affects the SQL Database's ability to restore from that point
					Out: false, // SQL Database changes don't affect the recoverable database itself (it's a point-in-time snapshot)
				}, // SQL Database depends on its recoverable database for disaster recovery and restore capabilities
			})
		}
	}

	if database.Properties != nil && database.Properties.RestorableDroppedDatabaseID != nil {
		// Extract server name and database name from RestorableDroppedDatabaseID resource ID
		// This handles cross-server scenarios where dropped databases may be on different servers
		// RestorableDroppedDatabaseID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Sql/servers/{serverName}/restorableDroppedDatabases/{databaseName}
		restorableDroppedServerName, restorableDroppedDatabaseName := azureshared.ExtractSQLRestorableDroppedDatabaseInfoFromResourceID(*database.Properties.RestorableDroppedDatabaseID)
		if restorableDroppedServerName != "" && restorableDroppedDatabaseName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLRestorableDroppedDatabase.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(restorableDroppedServerName, restorableDroppedDatabaseName),
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Restorable dropped database deletion/purge affects the SQL Database's ability to restore from that dropped database
					Out: false, // SQL Database changes don't affect the restorable dropped database itself (it's already dropped)
				}, // SQL Database depends on its restorable dropped database for restore capabilities after accidental deletion
			})
		}
	}

	if database.Properties != nil && database.Properties.RecoveryServicesRecoveryPointID != nil {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLRecoveryServicesRecoveryPoint.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // Recovery point deletion affects the SQL Database's ability to restore from that specific backup point
				Out: false, // SQL Database changes don't affect the recovery point itself (it's a point-in-time snapshot)
			}, // SQL Database depends on Recovery Services recovery points for backup and restore capabilities
		})
	}

	if database.Properties != nil && database.Properties.SourceDatabaseID != nil {
		// Extract server name and database name from SourceDatabaseID resource ID
		// This handles cross-server copy scenarios where the source database may be on a different server
		sourceServerName, sourceDatabaseName := azureshared.ExtractSQLDatabaseInfoFromResourceID(*database.Properties.SourceDatabaseID)
		if sourceServerName != "" && sourceDatabaseName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLDatabase.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(sourceServerName, sourceDatabaseName),
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  false, // Source database changes don't affect the copy (copy is independent after creation)
					Out: false, // Copy database changes don't affect the source database
				}, // Database copy is independent from its source after the copy operation completes
			})
		}
	}

	// Handle SourceResourceID - a generic resource ID that can reference different Azure resource types
	// When sourceResourceId is specified, it's used for PointInTimeRestore, Restore, or Recover operations
	// and can point to SQL databases, SQL elastic pools, or Synapse SQL pools
	if database.Properties != nil && database.Properties.SourceResourceID != nil {
		resourceType, params := azureshared.DetermineSourceResourceType(*database.Properties.SourceResourceID)

		switch resourceType {
		case azureshared.SourceResourceTypeSQLDatabase:
			serverName := params["serverName"]
			databaseName := params["databaseName"]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLDatabase.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(serverName, databaseName),
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Source database changes (especially deletion) affect the database's ability to restore
					Out: false, // Database changes don't affect the source database itself
				}, // SQL Database depends on the source SQL database for restore/recovery operations
			})

		case azureshared.SourceResourceTypeSQLElasticPool:
			elasticPoolName := params["elasticPoolName"]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLElasticPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  elasticPoolName,
					Scope:  s.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Source elastic pool changes (especially deletion) affect the database's ability to restore
					Out: false, // Database changes don't affect the source elastic pool itself
				}, // SQL Database depends on the source SQL elastic pool for restore/recovery operations
			})

		case azureshared.SourceResourceTypeUnknown:
			// Synapse SQL Pool and other resource types not yet supported
			// This could be extended in the future to support Synapse SQL pools
			// when Synapse item types are added to the codebase
		}
	}

	return sdpItem, nil
}

func (s sqlDatabaseWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLDatabaseLookupByName,
	}
}

func (s sqlDatabaseWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: serverName",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]

	pager := s.client.ListByServer(ctx, s.ResourceGroup(), serverName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
		}
		for _, database := range page.Value {
			if database.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlDatabaseToSDPItem(database, serverName, *database.Name)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlDatabaseWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			SQLServerLookupByName,
		},
	}
}

func (s sqlDatabaseWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.SQLServer:                        true,
		azureshared.SQLElasticPool:                   true,
		azureshared.SQLRecoverableDatabase:           true,
		azureshared.SQLRestorableDroppedDatabase:     true,
		azureshared.SQLRecoveryServicesRecoveryPoint: true,
	}
}

func (s sqlDatabaseWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_mssql_database.id",
		},
	}
}

func (s sqlDatabaseWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/databases/read",
	}
}

func (s sqlDatabaseWrapper) PredefinedRole() string {
	return "Reader"
}
