package shared

import (
	"fmt"
	"net/url"
	"regexp"
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
		"azure-storage-queue":                       {"storageAccounts", "queues"},
		"azure-storage-blob-container":              {"storageAccounts", "containers"},
		"azure-storage-file-share":                  {"storageAccounts", "shares"},
		"azure-storage-table":                       {"storageAccounts", "tables"},
		"azure-sql-database":                        {"servers", "databases"},           // "/subscriptions/00000000-1111-2222-3333-444444444444/resourceGroups/Default-SQL-SouthEastAsia/providers/Microsoft.Sql/servers/testsvr/databases/testdb",
		"azure-dbforpostgresql-database":            {"flexibleServers", "databases"},   // "/subscriptions/00000000-1111-2222-3333-444444444444/resourceGroups/Default-PostgreSQL-SouthEastAsia/providers/Microsoft.DBforPostgreSQL/flexibleServers/testsvr/databases/testdb",
		"azure-keyvault-secret":                     {"vaults", "secrets"},              // "/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{vaultName}/secrets/{secretName}",
		"azure-authorization-role-assignment":       {"roleAssignments"},                // "/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Authorization/roleAssignments/{roleAssignmentName}",
		"azure-compute-virtual-machine-run-command": {"virtualMachines", "runCommands"}, // "/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{virtualMachineName}/runCommands/{runCommandName}",
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

// ExtractSQLRecoverableDatabaseInfoFromResourceID extracts SQL server name and database name from a recoverable database resource ID.
// Azure SQL recoverable database IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/recoverableDatabases/{databaseName}
// Returns serverName and databaseName if the resource ID is a recoverable database, otherwise returns empty strings.
func ExtractSQLRecoverableDatabaseInfoFromResourceID(resourceID string) (serverName, databaseName string) {
	if resourceID == "" {
		return "", ""
	}

	params := ExtractPathParamsFromResourceID(resourceID, []string{"servers", "recoverableDatabases"})
	if len(params) >= 2 {
		return params[0], params[1]
	}

	return "", ""
}

// ExtractSQLRestorableDroppedDatabaseInfoFromResourceID extracts SQL server name and database name from a restorable dropped database resource ID.
// Azure SQL restorable dropped database IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/restorableDroppedDatabases/{databaseName}
// Returns serverName and databaseName if the resource ID is a restorable dropped database, otherwise returns empty strings.
func ExtractSQLRestorableDroppedDatabaseInfoFromResourceID(resourceID string) (serverName, databaseName string) {
	if resourceID == "" {
		return "", ""
	}

	params := ExtractPathParamsFromResourceID(resourceID, []string{"servers", "restorableDroppedDatabases"})
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

// ExtractVaultNameFromURI extracts the vault name from a Key Vault URI
// Format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
func ExtractVaultNameFromURI(uri string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	host := parsedURL.Host
	// Extract vault name from hostname: {vaultName}.vault.azure.net
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// ExtractKeyNameFromURI extracts the key name from a Key Vault key URI
// Format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
func ExtractKeyNameFromURI(uri string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	path := strings.Trim(parsedURL.Path, "/")
	parts := strings.Split(path, "/")
	// Path format: keys/{keyName}/{version}
	if len(parts) >= 2 && parts[0] == "keys" {
		return parts[1]
	}

	return ""
}

// ExtractSecretNameFromURI extracts the secret name from a Key Vault secret URI
// Format: https://{vaultName}.vault.azure.net/secrets/{secretName}/{version}
func ExtractSecretNameFromURI(uri string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	path := strings.Trim(parsedURL.Path, "/")
	parts := strings.Split(path, "/")
	// Path format: secrets/{secretName}/{version}
	if len(parts) >= 2 && parts[0] == "secrets" {
		return parts[1]
	}

	return ""
}

// ExtractSubscriptionIDFromResourceID extracts the subscription ID from an Azure resource ID
// Azure resource IDs follow the format:
// /subscriptions/{subscriptionId}/providers/... or
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/...
// This function returns just the subscription ID
// Returns empty string if the subscription ID cannot be found
func ExtractSubscriptionIDFromResourceID(resourceID string) string {
	if resourceID == "" {
		return ""
	}

	parts := strings.Split(strings.Trim(resourceID, "/"), "/")
	for i, part := range parts {
		if part == "subscriptions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// ExtractScopeFromResourceID extracts the scope (subscription.resourceGroup) from an Azure resource ID
// Azure resource IDs follow the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/...
// This function returns the scope in the format: "{subscriptionId}.{resourceGroupName}"
// Returns empty string if the resource ID doesn't contain both subscription and resource group
func ExtractScopeFromResourceID(resourceID string) string {
	if resourceID == "" {
		return ""
	}

	parts := strings.Split(strings.Trim(resourceID, "/"), "/")
	if len(parts) < 4 {
		return ""
	}

	// Find subscription ID (should be at index 1 after splitting)
	subscriptionID := ""
	resourceGroupName := ""

	for i, part := range parts {
		if part == "subscriptions" && i+1 < len(parts) {
			subscriptionID = parts[i+1]
		}
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroupName = parts[i+1]
		}
	}

	if subscriptionID != "" && resourceGroupName != "" {
		return fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
	}

	return ""
}

// ExtractDNSFromURL extracts the DNS name from a URL
// Example: https://account.blob.core.windows.net/ -> account.blob.core.windows.net
func ExtractDNSFromURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	// Remove protocol prefix (http:// or https://)
	if idx := len("https://"); len(urlStr) > idx && urlStr[:idx] == "https://" {
		urlStr = urlStr[idx:]
	} else if idx := len("http://"); len(urlStr) > idx && urlStr[:idx] == "http://" {
		urlStr = urlStr[idx:]
	}
	// Remove trailing slash and path
	if idx := len(urlStr); idx > 0 && urlStr[idx-1] == '/' {
		urlStr = urlStr[:idx-1]
	}
	// Extract hostname (everything before the first /)
	if idx := len(urlStr); idx > 0 {
		for i := range idx {
			if urlStr[i] == '/' {
				urlStr = urlStr[:i]
				break
			}
		}
	}
	return urlStr
}

// ExtractStorageAccountNameFromBlobURI extracts the storage account name from an Azure blob URI
// Blob URIs follow the format: https://{accountName}.blob.core.windows.net/{container}/{blob}
func ExtractStorageAccountNameFromBlobURI(blobURI string) string {
	if blobURI == "" {
		return ""
	}

	parsedURL, err := url.Parse(blobURI)
	if err != nil {
		return ""
	}

	host := parsedURL.Host
	// Extract account name from hostname: {accountName}.blob.core.windows.net
	parts := strings.Split(host, ".")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return ""
}

// ExtractContainerNameFromBlobURI extracts the container name from an Azure blob URI
// Blob URIs follow the format: https://{accountName}.blob.core.windows.net/{container}/{blob}
// Returns the first path segment which is the container name
func ExtractContainerNameFromBlobURI(blobURI string) string {
	if blobURI == "" {
		return ""
	}

	// Defensive check: ensure this is actually a blob URI
	if !strings.Contains(blobURI, ".blob.core.windows.net") {
		return ""
	}

	parsedURL, err := url.Parse(blobURI)
	if err != nil {
		return ""
	}

	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return ""
	}

	// Split path and get the first segment (container name)
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return ""
}

// ref: https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/get?view=rest-authorization-2022-04-01&tabs=HTTP
// subscriptionIDPattern matches Azure subscription IDs (UUID format: 8-4-4-4-12 hex digits)
var subscriptionIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ConstructRoleAssignmentScope constructs an Azure role assignment scope path from a scope input.
// The scopeInput is usually in the format "{subscriptionId}.{resourceGroup}".
// If the input contains a dot, it's split into subscription ID and resource group name.
// If the input matches a UUID pattern (no dot), it's treated as a subscription ID.
// Otherwise, it's treated as a resource group name and uses the provided subscriptionID parameter.
//
// Parameters:
//   - scopeInput: Usually in format "{subscriptionId}.{resourceGroup}", or a subscription ID (UUID), or a resource group name
//   - subscriptionID: The subscription ID to use when constructing resource group scopes (fallback when scopeInput is just a resource group name)
//
// Returns:
//   - The Azure scope path in the format expected by the Azure SDK
func ConstructRoleAssignmentScope(scopeInput, subscriptionID string) string {
	if scopeInput == "" {
		return ""
	}

	// Check if scopeInput is in the format "{subscriptionId}.{resourceGroup}"
	if strings.Contains(scopeInput, ".") {
		parts := strings.SplitN(scopeInput, ".", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			// It's in the format subscriptionId.resourceGroup - construct resource group scope
			return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", parts[0], parts[1])
		}
	}

	// Check if scopeInput is a subscription ID (UUID format)
	if subscriptionIDPattern.MatchString(scopeInput) {
		// It's a subscription ID - construct subscription scope
		return "/subscriptions/" + scopeInput
	}

	// It's a resource group name - construct resource group scope using provided subscriptionID
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, scopeInput)
}
