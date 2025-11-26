package shared

import (
	"strings"
)

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
