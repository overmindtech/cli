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
