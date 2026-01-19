package shared_test

import (
	"reflect"
	"testing"

	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func TestGetResourceIDPathKeys(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expected     []string
	}{
		{
			name:         "storage queue",
			resourceType: "azure-storage-queue",
			expected:     []string{"storageAccounts", "queues"},
		},
		{
			name:         "storage blob container",
			resourceType: "azure-storage-blob-container",
			expected:     []string{"storageAccounts", "containers"},
		},
		{
			name:         "storage file share",
			resourceType: "azure-storage-file-share",
			expected:     []string{"storageAccounts", "shares"},
		},
		{
			name:         "storage table",
			resourceType: "azure-storage-table",
			expected:     []string{"storageAccounts", "tables"},
		},
		{
			name:         "unknown resource type",
			resourceType: "azure-unknown-resource",
			expected:     nil,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			expected:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.GetResourceIDPathKeys(tc.resourceType)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("GetResourceIDPathKeys(%q) = %v; want %v", tc.resourceType, actual, tc.expected)
			}
		})
	}
}

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "valid storage account resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount",
			expected:   "teststorageaccount",
		},
		{
			name:       "valid storage queue resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			expected:   "test-queue",
		},
		{
			name:       "valid compute disk resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Compute/disks/test-disk",
			expected:   "test-disk",
		},
		{
			name:       "resource ID with trailing slash",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/",
			expected:   "",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expected:   "",
		},
		{
			name:       "single segment",
			resourceID: "resource-name",
			expected:   "resource-name",
		},
		{
			name:       "resource ID starting with slash",
			resourceID: "/resource-name",
			expected:   "resource-name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractResourceName(tc.resourceID)
			if actual != tc.expected {
				t.Errorf("ExtractResourceName(%q) = %q; want %q", tc.resourceID, actual, tc.expected)
			}
		})
	}
}

func TestExtractPathParamsFromResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		keys       []string
		expected   []string
	}{
		{
			name:       "storage queue - extract storage account and queue",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       []string{"storageAccounts", "queues"},
			expected:   []string{"teststorageaccount", "test-queue"},
		},
		{
			name:       "storage blob container - extract storage account and container",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/blobServices/default/containers/my-container",
			keys:       []string{"storageAccounts", "containers"},
			expected:   []string{"teststorageaccount", "my-container"},
		},
		{
			name:       "storage file share - extract storage account and share",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/fileServices/default/shares/my-share",
			keys:       []string{"storageAccounts", "shares"},
			expected:   []string{"teststorageaccount", "my-share"},
		},
		{
			name:       "storage table - extract storage account and table",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/tableServices/default/tables/my-table",
			keys:       []string{"storageAccounts", "tables"},
			expected:   []string{"teststorageaccount", "my-table"},
		},
		{
			name:       "single key extraction",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount",
			keys:       []string{"storageAccounts"},
			expected:   []string{"teststorageaccount"},
		},
		{
			name:       "resource ID without leading slash",
			resourceID: "subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       []string{"storageAccounts", "queues"},
			expected:   []string{"teststorageaccount", "test-queue"},
		},
		{
			name:       "keys not in order",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       []string{"queues", "storageAccounts"},
			expected:   []string{"test-queue", "teststorageaccount"},
		},
		{
			name:       "missing key",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       []string{"storageAccounts", "missingKey"},
			expected:   nil,
		},
		{
			name:       "key exists but no value after it",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts",
			keys:       []string{"storageAccounts"},
			expected:   nil,
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			keys:       []string{"storageAccounts", "queues"},
			expected:   nil,
		},
		{
			name:       "empty keys",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       []string{},
			expected:   nil,
		},
		{
			name:       "nil keys",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue",
			keys:       nil,
			expected:   nil,
		},
		{
			name:       "resource ID with trailing slash",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/test-queue/",
			keys:       []string{"storageAccounts", "queues"},
			expected:   []string{"teststorageaccount", "test-queue"},
		},
		{
			name:       "duplicate keys - returns first occurrence",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/first-account/queueServices/default/storageAccounts/second-account/queues/test-queue",
			keys:       []string{"storageAccounts", "queues"},
			expected:   []string{"first-account", "test-queue"},
		},
		{
			name:       "keys with special characters in values",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-storage-account_123/queueServices/default/queues/test_queue-name",
			keys:       []string{"storageAccounts", "queues"},
			expected:   []string{"test-storage-account_123", "test_queue-name"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractPathParamsFromResourceID(tc.resourceID, tc.keys)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("ExtractPathParamsFromResourceID(%q, %v) = %v; want %v", tc.resourceID, tc.keys, actual, tc.expected)
			}
		})
	}
}

func TestConvertAzureTags(t *testing.T) {
	tests := []struct {
		name      string
		azureTags map[string]*string
		expected  map[string]string
	}{
		{
			name: "valid tags with values",
			azureTags: map[string]*string{
				"env":     stringPtr("production"),
				"project": stringPtr("overmind"),
				"team":    stringPtr("platform"),
			},
			expected: map[string]string{
				"env":     "production",
				"project": "overmind",
				"team":    "platform",
			},
		},
		{
			name:      "nil tags",
			azureTags: nil,
			expected:  nil,
		},
		{
			name:      "empty tags",
			azureTags: map[string]*string{},
			expected:  map[string]string{},
		},
		{
			name: "tags with nil values - should be skipped",
			azureTags: map[string]*string{
				"env":     stringPtr("production"),
				"project": nil,
				"team":    stringPtr("platform"),
			},
			expected: map[string]string{
				"env":  "production",
				"team": "platform",
			},
		},
		{
			name: "all nil values",
			azureTags: map[string]*string{
				"env":     nil,
				"project": nil,
				"team":    nil,
			},
			expected: map[string]string{},
		},
		{
			name: "single tag",
			azureTags: map[string]*string{
				"env": stringPtr("test"),
			},
			expected: map[string]string{
				"env": "test",
			},
		},
		{
			name: "tags with empty string values",
			azureTags: map[string]*string{
				"env":     stringPtr(""),
				"project": stringPtr("overmind"),
			},
			expected: map[string]string{
				"env":     "",
				"project": "overmind",
			},
		},
		{
			name: "tags with special characters",
			azureTags: map[string]*string{
				"tag-with-dashes": stringPtr("value_with_underscores"),
				"tag.with.dots":   stringPtr("value with spaces"),
			},
			expected: map[string]string{
				"tag-with-dashes": "value_with_underscores",
				"tag.with.dots":   "value with spaces",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ConvertAzureTags(tc.azureTags)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("ConvertAzureTags(%v) = %v; want %v", tc.azureTags, actual, tc.expected)
			}
		})
	}
}

// stringPtr is a helper function to create a pointer to a string
func stringPtr(s string) *string {
	return &s
}

func TestExtractSQLServerNameFromDatabaseID(t *testing.T) {
	tests := []struct {
		name       string
		databaseID string
		expected   string
	}{
		{
			name:       "valid SQL database resource ID",
			databaseID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/databases/test-db",
			expected:   "test-server",
		},
		{
			name:       "empty database ID",
			databaseID: "",
			expected:   "",
		},
		{
			name:       "invalid resource ID format",
			databaseID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "",
		},
		{
			name:       "resource ID without servers segment",
			databaseID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/databases/test-db",
			expected:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractSQLServerNameFromDatabaseID(tc.databaseID)
			if actual != tc.expected {
				t.Errorf("ExtractSQLServerNameFromDatabaseID(%q) = %q; want %q", tc.databaseID, actual, tc.expected)
			}
		})
	}
}

func TestExtractSQLElasticPoolNameFromID(t *testing.T) {
	tests := []struct {
		name         string
		elasticPoolID string
		expected     string
	}{
		{
			name:         "valid SQL elastic pool resource ID",
			elasticPoolID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/elasticPools/test-pool",
			expected:     "test-pool",
		},
		{
			name:         "empty elastic pool ID",
			elasticPoolID: "",
			expected:     "",
		},
		{
			name:         "invalid resource ID format",
			elasticPoolID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:     "",
		},
		{
			name:         "resource ID without elasticPools segment",
			elasticPoolID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server",
			expected:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractSQLElasticPoolNameFromID(tc.elasticPoolID)
			if actual != tc.expected {
				t.Errorf("ExtractSQLElasticPoolNameFromID(%q) = %q; want %q", tc.elasticPoolID, actual, tc.expected)
			}
		})
	}
}

func TestExtractSQLDatabaseInfoFromResourceID(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		expectedServer string
		expectedDB    string
	}{
		{
			name:           "valid SQL database resource ID",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/databases/test-db",
			expectedServer: "test-server",
			expectedDB:    "test-db",
		},
		{
			name:           "empty resource ID",
			resourceID:     "",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "invalid resource ID format",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "resource ID missing databases segment",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server",
			expectedServer: "",
			expectedDB:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualServer, actualDB := azureshared.ExtractSQLDatabaseInfoFromResourceID(tc.resourceID)
			if actualServer != tc.expectedServer {
				t.Errorf("ExtractSQLDatabaseInfoFromResourceID(%q) server = %q; want %q", tc.resourceID, actualServer, tc.expectedServer)
			}
			if actualDB != tc.expectedDB {
				t.Errorf("ExtractSQLDatabaseInfoFromResourceID(%q) database = %q; want %q", tc.resourceID, actualDB, tc.expectedDB)
			}
		})
	}
}

func TestExtractSQLRecoverableDatabaseInfoFromResourceID(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		expectedServer string
		expectedDB    string
	}{
		{
			name:           "valid recoverable database resource ID",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/recoverableDatabases/test-db",
			expectedServer: "test-server",
			expectedDB:    "test-db",
		},
		{
			name:           "empty resource ID",
			resourceID:     "",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "invalid resource ID format",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "resource ID missing recoverableDatabases segment",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server",
			expectedServer: "",
			expectedDB:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualServer, actualDB := azureshared.ExtractSQLRecoverableDatabaseInfoFromResourceID(tc.resourceID)
			if actualServer != tc.expectedServer {
				t.Errorf("ExtractSQLRecoverableDatabaseInfoFromResourceID(%q) server = %q; want %q", tc.resourceID, actualServer, tc.expectedServer)
			}
			if actualDB != tc.expectedDB {
				t.Errorf("ExtractSQLRecoverableDatabaseInfoFromResourceID(%q) database = %q; want %q", tc.resourceID, actualDB, tc.expectedDB)
			}
		})
	}
}

func TestExtractSQLRestorableDroppedDatabaseInfoFromResourceID(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		expectedServer string
		expectedDB    string
	}{
		{
			name:           "valid restorable dropped database resource ID",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/restorableDroppedDatabases/test-db",
			expectedServer: "test-server",
			expectedDB:    "test-db",
		},
		{
			name:           "empty resource ID",
			resourceID:     "",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "invalid resource ID format",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expectedServer: "",
			expectedDB:    "",
		},
		{
			name:           "resource ID missing restorableDroppedDatabases segment",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server",
			expectedServer: "",
			expectedDB:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualServer, actualDB := azureshared.ExtractSQLRestorableDroppedDatabaseInfoFromResourceID(tc.resourceID)
			if actualServer != tc.expectedServer {
				t.Errorf("ExtractSQLRestorableDroppedDatabaseInfoFromResourceID(%q) server = %q; want %q", tc.resourceID, actualServer, tc.expectedServer)
			}
			if actualDB != tc.expectedDB {
				t.Errorf("ExtractSQLRestorableDroppedDatabaseInfoFromResourceID(%q) database = %q; want %q", tc.resourceID, actualDB, tc.expectedDB)
			}
		})
	}
}

func TestExtractSQLElasticPoolInfoFromResourceID(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		expectedServer string
		expectedPool string
	}{
		{
			name:           "valid SQL elastic pool resource ID",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/elasticPools/test-pool",
			expectedServer: "test-server",
			expectedPool:  "test-pool",
		},
		{
			name:           "empty resource ID",
			resourceID:     "",
			expectedServer: "",
			expectedPool:  "",
		},
		{
			name:           "invalid resource ID format",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expectedServer: "",
			expectedPool:  "",
		},
		{
			name:           "resource ID missing elasticPools segment",
			resourceID:     "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server",
			expectedServer: "",
			expectedPool:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualServer, actualPool := azureshared.ExtractSQLElasticPoolInfoFromResourceID(tc.resourceID)
			if actualServer != tc.expectedServer {
				t.Errorf("ExtractSQLElasticPoolInfoFromResourceID(%q) server = %q; want %q", tc.resourceID, actualServer, tc.expectedServer)
			}
			if actualPool != tc.expectedPool {
				t.Errorf("ExtractSQLElasticPoolInfoFromResourceID(%q) pool = %q; want %q", tc.resourceID, actualPool, tc.expectedPool)
			}
		})
	}
}

func TestDetermineSourceResourceType(t *testing.T) {
	tests := []struct {
		name           string
		resourceID     string
		expectedType   azureshared.SourceResourceType
		expectedParams map[string]string
	}{
		{
			name:       "SQL database resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/databases/test-db",
			expectedType: azureshared.SourceResourceTypeSQLDatabase,
			expectedParams: map[string]string{
				"serverName":   "test-server",
				"databaseName": "test-db",
			},
		},
		{
			name:       "SQL elastic pool resource ID",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/elasticPools/test-pool",
			expectedType: azureshared.SourceResourceTypeSQLElasticPool,
			expectedParams: map[string]string{
				"serverName":      "test-server",
				"elasticPoolName": "test-pool",
			},
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expectedType: azureshared.SourceResourceTypeUnknown,
			expectedParams: nil,
		},
		{
			name:       "unknown resource type",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expectedType: azureshared.SourceResourceTypeUnknown,
			expectedParams: nil,
		},
		{
			name:       "Synapse SQL pool (not yet supported)",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Synapse/workspaces/test-workspace/sqlPools/test-pool",
			expectedType: azureshared.SourceResourceTypeUnknown,
			expectedParams: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualType, actualParams := azureshared.DetermineSourceResourceType(tc.resourceID)
			if actualType != tc.expectedType {
				t.Errorf("DetermineSourceResourceType(%q) type = %v; want %v", tc.resourceID, actualType, tc.expectedType)
			}
			if !reflect.DeepEqual(actualParams, tc.expectedParams) {
				t.Errorf("DetermineSourceResourceType(%q) params = %v; want %v", tc.resourceID, actualParams, tc.expectedParams)
			}
		})
	}
}

func TestExtractVaultNameFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid Key Vault key URI",
			uri:      "https://test-vault.vault.azure.net/keys/test-key/version",
			expected: "test-vault",
		},
		{
			name:     "valid Key Vault secret URI",
			uri:      "https://my-vault.vault.azure.net/secrets/my-secret/version",
			expected: "my-vault",
		},
		{
			name:     "vault name with hyphens",
			uri:      "https://test-vault-name.vault.azure.net/keys/test-key/version",
			expected: "test-vault-name",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "invalid URI format",
			uri:      "not-a-valid-uri",
			expected: "",
		},
		{
			name:     "URI without vault domain",
			uri:      "https://example.com/path",
			expected: "example",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractVaultNameFromURI(tc.uri)
			if actual != tc.expected {
				t.Errorf("ExtractVaultNameFromURI(%q) = %q; want %q", tc.uri, actual, tc.expected)
			}
		})
	}
}

func TestExtractKeyNameFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid Key Vault key URI",
			uri:      "https://test-vault.vault.azure.net/keys/test-key/version",
			expected: "test-key",
		},
		{
			name:     "key name with hyphens",
			uri:      "https://test-vault.vault.azure.net/keys/my-test-key-name/version",
			expected: "my-test-key-name",
		},
		{
			name:     "key URI without version",
			uri:      "https://test-vault.vault.azure.net/keys/test-key",
			expected: "test-key",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "invalid URI format",
			uri:      "not-a-valid-uri",
			expected: "",
		},
		{
			name:     "URI for secret (not key)",
			uri:      "https://test-vault.vault.azure.net/secrets/test-secret/version",
			expected: "",
		},
		{
			name:     "URI without keys path",
			uri:      "https://test-vault.vault.azure.net/",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractKeyNameFromURI(tc.uri)
			if actual != tc.expected {
				t.Errorf("ExtractKeyNameFromURI(%q) = %q; want %q", tc.uri, actual, tc.expected)
			}
		})
	}
}

func TestExtractSecretNameFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "valid Key Vault secret URI",
			uri:      "https://test-vault.vault.azure.net/secrets/test-secret/version",
			expected: "test-secret",
		},
		{
			name:     "secret name with hyphens",
			uri:      "https://test-vault.vault.azure.net/secrets/my-test-secret-name/version",
			expected: "my-test-secret-name",
		},
		{
			name:     "secret URI without version",
			uri:      "https://test-vault.vault.azure.net/secrets/test-secret",
			expected: "test-secret",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "invalid URI format",
			uri:      "not-a-valid-uri",
			expected: "",
		},
		{
			name:     "URI for key (not secret)",
			uri:      "https://test-vault.vault.azure.net/keys/test-key/version",
			expected: "",
		},
		{
			name:     "URI without secrets path",
			uri:      "https://test-vault.vault.azure.net/",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractSecretNameFromURI(tc.uri)
			if actual != tc.expected {
				t.Errorf("ExtractSecretNameFromURI(%q) = %q; want %q", tc.uri, actual, tc.expected)
			}
		})
	}
}

func TestExtractScopeFromResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "valid resource ID with subscription and resource group",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "12345678-1234-1234-1234-123456789012.test-rg",
		},
		{
			name:       "resource ID with nested resources",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account/queueServices/default/queues/test-queue",
			expected:   "12345678-1234-1234-1234-123456789012.test-rg",
		},
		{
			name:       "resource ID without leading slash",
			resourceID: "subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "12345678-1234-1234-1234-123456789012.test-rg",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			expected:   "",
		},
		{
			name:       "resource ID missing subscription",
			resourceID: "/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "",
		},
		{
			name:       "resource ID missing resource group",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "",
		},
		{
			name:       "resource ID too short",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012",
			expected:   "",
		},
		{
			name:       "resource ID with subscription but no resource group value (malformed - would not occur in practice)",
			resourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/providers/Microsoft.Storage/storageAccounts/test-account",
			expected:   "12345678-1234-1234-1234-123456789012.providers",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractScopeFromResourceID(tc.resourceID)
			if actual != tc.expected {
				t.Errorf("ExtractScopeFromResourceID(%q) = %q; want %q", tc.resourceID, actual, tc.expected)
			}
		})
	}
}

func TestExtractDNSFromURL(t *testing.T) {
	tests := []struct {
		name     string
		urlStr   string
		expected string
	}{
		{
			name:     "HTTPS URL with path",
			urlStr:   "https://account.blob.core.windows.net/container/blob",
			expected: "account.blob.core.windows.net",
		},
		{
			name:     "HTTPS URL without path",
			urlStr:   "https://account.blob.core.windows.net",
			expected: "account.blob.core.windows.net",
		},
		{
			name:     "HTTPS URL with trailing slash",
			urlStr:   "https://account.blob.core.windows.net/",
			expected: "account.blob.core.windows.net",
		},
		{
			name:     "HTTP URL",
			urlStr:   "http://example.com/path/to/resource",
			expected: "example.com",
		},
		{
			name:     "HTTP URL without path",
			urlStr:   "http://example.com",
			expected: "example.com",
		},
		{
			name:     "URL with port",
			urlStr:   "https://example.com:8080/path",
			expected: "example.com:8080",
		},
		{
			name:     "empty URL",
			urlStr:   "",
			expected: "",
		},
		{
			name:     "URL without protocol",
			urlStr:   "example.com/path",
			expected: "example.com",
		},
		{
			name:     "URL with query parameters",
			urlStr:   "https://example.com/path?param=value",
			expected: "example.com",
		},
		{
			name:     "URL with fragment",
			urlStr:   "https://example.com/path#fragment",
			expected: "example.com",
		},
		{
			name:     "complex storage account URL",
			urlStr:   "https://mystorageaccount.blob.core.windows.net/mycontainer/myblob.txt",
			expected: "mystorageaccount.blob.core.windows.net",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := azureshared.ExtractDNSFromURL(tc.urlStr)
			if actual != tc.expected {
				t.Errorf("ExtractDNSFromURL(%q) = %q; want %q", tc.urlStr, actual, tc.expected)
			}
		})
	}
}
