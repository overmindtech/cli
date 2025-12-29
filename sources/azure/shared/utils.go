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
