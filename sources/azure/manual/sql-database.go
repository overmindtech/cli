package manual

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var SQLDatabaseLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLDatabase)

type sqlDatabaseWrapper struct {
	client clients.SqlDatabasesClient

	*azureshared.MultiResourceGroupBase
}

func NewSqlDatabase(client clients.SqlDatabasesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &sqlDatabaseWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLDatabase,
		),
	}
}

func (s sqlDatabaseWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: serverName and databaseName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	serverName := queryParts[0]
	databaseName := queryParts[1]

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, serverName, databaseName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureSqlDatabaseToSDPItem(&resp.Database, serverName, databaseName, scope)
}

func (s sqlDatabaseWrapper) azureSqlDatabaseToSDPItem(database *armsql.Database, serverName, databaseName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(database, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(serverName, databaseName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLDatabase.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
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
					Scope:  scope,
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
					Scope:  scope,
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
					Scope:  scope,
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
					Scope:  scope,
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
				Scope:  scope,
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
					Scope:  scope,
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
					Scope:  scope,
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
					Scope:  scope,
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

	if database.Properties != nil && database.Properties.FailoverGroupID != nil {
		// FailoverGroupID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Sql/servers/{serverName}/failoverGroups/{failoverGroupName}
		params := azureshared.ExtractPathParamsFromResourceID(*database.Properties.FailoverGroupID, []string{"servers", "failoverGroups"})
		if len(params) >= 2 {
			failoverServerName := params[0]
			failoverGroupName := params[1]
			linkedScope := azureshared.ExtractScopeFromResourceID(*database.Properties.FailoverGroupID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLServerFailoverGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(failoverServerName, failoverGroupName),
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Failover group deletion or failover affects the database's availability and replication
					Out: false, // Database membership in the group doesn't change the failover group configuration
				}, // SQL Database belongs to a Failover Group for high availability
			})
		}
	}

	if database.Properties != nil && database.Properties.LongTermRetentionBackupResourceID != nil {
		locationName, ltrServerName, ltrDatabaseName, backupName := azureshared.ExtractSQLLongTermRetentionBackupInfoFromResourceID(*database.Properties.LongTermRetentionBackupResourceID)
		if locationName != "" && ltrServerName != "" && ltrDatabaseName != "" && backupName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(*database.Properties.LongTermRetentionBackupResourceID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.SQLLongTermRetentionBackup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(locationName, ltrServerName, ltrDatabaseName, backupName),
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // LTR backup deletion affects the database's ability to restore from that backup
					Out: false, // SQL Database changes don't affect the LTR backup itself
				}, // SQL Database depends on LTR backup for long-term retention restore
			})
		}
	}

	if database.Properties != nil && database.Properties.MaintenanceConfigurationID != nil && *database.Properties.MaintenanceConfigurationID != "" {
		configName := azureshared.ExtractResourceName(*database.Properties.MaintenanceConfigurationID)
		if configName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(*database.Properties.MaintenanceConfigurationID)
			if linkedScope == "" && strings.Contains(*database.Properties.MaintenanceConfigurationID, "publicMaintenanceConfigurations") {
				linkedScope = azureshared.ExtractSubscriptionIDFromResourceID(*database.Properties.MaintenanceConfigurationID)
			}
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.MaintenanceMaintenanceConfiguration.String(),
					Method: sdp.QueryMethod_GET,
					Query:  configName,
					Scope:  linkedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Maintenance config changes affect when maintenance updates occur for the database
					Out: false, // Database changes don't affect the maintenance configuration itself
				}, // SQL Database uses Maintenance Configuration for update scheduling
			})
		}
	}

	// Link Key Vault Keys from EncryptionProtector and Keys map (deduplicate by vaultName+keyName)
	seenKeyVaultKeys := make(map[string]bool)
	addKeyVaultKeyLink := func(vaultName, keyName string) {
		if vaultName == "" || keyName == "" {
			return
		}
		key := vaultName + "|" + keyName
		if seenKeyVaultKeys[key] {
			return
		}
		seenKeyVaultKeys[key] = true
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.KeyVaultKey.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(vaultName, keyName),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // Key Vault Key deletion/rotation affects database encryption
				Out: false, // Database changes don't affect the Key Vault Key
			}, // SQL Database uses Key Vault Key for per-database CMK and encryption at rest
		})
	}
	if database.Properties != nil && database.Properties.EncryptionProtector != nil && *database.Properties.EncryptionProtector != "" {
		addKeyVaultKeyLink(
			azureshared.ExtractVaultNameFromURI(*database.Properties.EncryptionProtector),
			azureshared.ExtractKeyNameFromURI(*database.Properties.EncryptionProtector),
		)
	}
	if database.Properties != nil && database.Properties.Keys != nil {
		for keyURI := range database.Properties.Keys {
			addKeyVaultKeyLink(
				azureshared.ExtractVaultNameFromURI(keyURI),
				azureshared.ExtractKeyNameFromURI(keyURI),
			)
		}
	}

	if database.Identity != nil && database.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range database.Identity.UserAssignedIdentities {
			if identityResourceID == "" {
				continue
			}
			identityName := azureshared.ExtractResourceName(identityResourceID)
			linkedScope := azureshared.ExtractScopeFromResourceID(identityResourceID)
			if identityName != "" && linkedScope != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // User Assigned Identity deletion affects database identity and CMK access
						Out: false, // Database changes don't affect the User Assigned Identity
					}, // SQL Database uses User Assigned Identity for Azure AD auth and CMK
				})
			}
		}
	}

	// Database Schemas - child resource with LIST endpoint
	// GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}/schemas
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.SQLDatabaseSchema.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(serverName, databaseName),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Schema changes don't affect the parent database resource
			Out: true,  // Database deletion removes all schemas
		}, // Database Schemas are child resources of the SQL Database
	})

	return sdpItem, nil
}

func (s sqlDatabaseWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
		SQLDatabaseLookupByName,
	}
}

func (s sqlDatabaseWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, database := range page.Value {
			if database.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlDatabaseToSDPItem(database, serverName, *database.Name, scope)
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
		azureshared.SQLServer:                           true,
		azureshared.SQLDatabase:                         true, // source database / copy source
		azureshared.SQLElasticPool:                      true,
		azureshared.SQLRecoverableDatabase:              true,
		azureshared.SQLRestorableDroppedDatabase:        true,
		azureshared.SQLRecoveryServicesRecoveryPoint:    true,
		azureshared.SQLServerFailoverGroup:              true,
		azureshared.SQLLongTermRetentionBackup:          true,
		azureshared.MaintenanceMaintenanceConfiguration: true,
		azureshared.KeyVaultKey:                         true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		azureshared.SQLDatabaseSchema:                   true,
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
