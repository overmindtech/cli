package shared

import (
	"strings"
)

// GetResourceIDPathKeys returns the path keys to extract from an Azure resource ID
// for a given resource type. These keys are used to extract the necessary parameters
// from the resource ID to match the adapter's GetLookups() order.
//
// For example, for storage queues:
// Resource ID: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{account}/queueServices/default/queues/{queue}
// Path keys: ["storageAccounts", "queues"]
// Returns: ["{account}", "{queue}"]
func GetResourceIDPathKeys(resourceType string) []string {
	// Map of resource types to their path keys in the order they appear in GetLookups()
	pathKeysMap := map[string][]string{
		"azure-storage-queue":          {"storageAccounts", "queues"},
		"azure-storage-blob-container": {"storageAccounts", "containers"},
		"azure-storage-file-share":     {"storageAccounts", "shares"},
		"azure-storage-table":          {"storageAccounts", "tables"},
		"azure-sql-database":           {"servers", "databases"}, // "/subscriptions/00000000-1111-2222-3333-444444444444/resourceGroups/Default-SQL-SouthEastAsia/providers/Microsoft.Sql/servers/testsvr/databases/testdb",
	}

	if keys, ok := pathKeysMap[resourceType]; ok {
		return keys
	}

	return nil
}

// ExtractResourceName extracts the resource name from an Azure resource ID
// Azure resource IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProvider}/{resourceType}/{resourceName}
// This function returns the last segment of the path, which is typically the resource name
func ExtractResourceName(resourceID string) string {
	if resourceID == "" {
		return ""
	}

	// Split by "/" and get the last part (resource name)
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// ExtractPathParamsFromResourceID extracts values following specified path keys from an Azure resource ID.
// It returns a slice of values in the order of the keys provided.
//
// For example, for input="/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{account}/queueServices/default/queues/{queue}"
// and keys=["storageAccounts", "queues"], it will return ["{account}", "{queue}"].
//
// If a key is not found, the function will return nil.
func ExtractPathParamsFromResourceID(resourceID string, keys []string) []string {
	if resourceID == "" || len(keys) == 0 {
		return nil
	}

	parts := strings.Split(strings.Trim(resourceID, "/"), "/")
	results := make([]string, 0, len(keys))

	for _, key := range keys {
		found := false
		for i, part := range parts {
			if part == key && i+1 < len(parts) {
				results = append(results, parts[i+1])
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	if len(results) != len(keys) {
		return nil
	}

	return results
}

// ExtractSQLServerNameFromDatabaseID extracts the SQL server name from a SQL database resource ID.
// Azure SQL database IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}
// This function returns the server name segment.
func ExtractSQLServerNameFromDatabaseID(databaseID string) string {
	if databaseID == "" {
		return ""
	}

	params := ExtractPathParamsFromResourceID(databaseID, []string{"servers"})
	if len(params) > 0 {
		return params[0]
	}

	return ""
}

// ExtractSQLElasticPoolNameFromID extracts the SQL elastic pool name from an elastic pool resource ID.
// Azure SQL elastic pool IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/elasticPools/{elasticPoolName}
// This function returns the elastic pool name segment.
func ExtractSQLElasticPoolNameFromID(elasticPoolID string) string {
	if elasticPoolID == "" {
		return ""
	}

	params := ExtractPathParamsFromResourceID(elasticPoolID, []string{"elasticPools"})
	if len(params) > 0 {
		return params[0]
	}

	return ""
}

// ExtractSQLDatabaseInfoFromResourceID extracts SQL server name and database name from a SQL database resource ID.
// Azure SQL database IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}
// Returns serverName and databaseName if the resource ID is a SQL database, otherwise returns empty strings.
func ExtractSQLDatabaseInfoFromResourceID(resourceID string) (serverName, databaseName string) {
	if resourceID == "" {
		return "", ""
	}

	params := ExtractPathParamsFromResourceID(resourceID, []string{"servers", "databases"})
	if len(params) >= 2 {
		return params[0], params[1]
	}

	return "", ""
}

// ExtractSQLElasticPoolInfoFromResourceID extracts SQL server name and elastic pool name from a SQL elastic pool resource ID.
// Azure SQL elastic pool IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/elasticPools/{elasticPoolName}
// Returns serverName and elasticPoolName if the resource ID is a SQL elastic pool, otherwise returns empty strings.
func ExtractSQLElasticPoolInfoFromResourceID(resourceID string) (serverName, elasticPoolName string) {
	if resourceID == "" {
		return "", ""
	}

	params := ExtractPathParamsFromResourceID(resourceID, []string{"servers", "elasticPools"})
	if len(params) >= 2 {
		return params[0], params[1]
	}

	return "", ""
}

// SourceResourceType represents the type of resource referenced by SourceResourceID
type SourceResourceType int

const (
	SourceResourceTypeUnknown SourceResourceType = iota
	SourceResourceTypeSQLDatabase
	SourceResourceTypeSQLElasticPool
	// SourceResourceTypeSynapseSQLPool - not yet supported (requires Synapse item types)
)

// DetermineSourceResourceType determines the type of resource from a SourceResourceID.
// Returns the resource type and extracted parameters for SQL resources.
func DetermineSourceResourceType(resourceID string) (SourceResourceType, map[string]string) {
	if resourceID == "" {
		return SourceResourceTypeUnknown, nil
	}

	// Check for SQL Database
	if serverName, databaseName := ExtractSQLDatabaseInfoFromResourceID(resourceID); serverName != "" && databaseName != "" {
		return SourceResourceTypeSQLDatabase, map[string]string{
			"serverName":   serverName,
			"databaseName": databaseName,
		}
	}

	// Check for SQL Elastic Pool
	if serverName, poolName := ExtractSQLElasticPoolInfoFromResourceID(resourceID); serverName != "" && poolName != "" {
		return SourceResourceTypeSQLElasticPool, map[string]string{
			"serverName":      serverName,
			"elasticPoolName": poolName,
		}
	}

	// Check for Synapse SQL Pool (for future support)
	// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Synapse/workspaces/{workspaceName}/sqlPools/{poolName}
	params := ExtractPathParamsFromResourceID(resourceID, []string{"workspaces", "sqlPools"})
	if len(params) >= 2 {
		// Synapse not yet supported - return unknown for now
		return SourceResourceTypeUnknown, nil
	}

	return SourceResourceTypeUnknown, nil
}

// convertAzureTags converts Azure tags (map[string]*string) to SDP tags (map[string]string)
func ConvertAzureTags(azureTags map[string]*string) map[string]string {
	if azureTags == nil {
		return nil
	}

	tags := make(map[string]string, len(azureTags))
	for k, v := range azureTags {
		if v != nil {
			tags[k] = *v
		}
	}
	return tags
}
