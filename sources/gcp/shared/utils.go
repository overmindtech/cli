package shared

import (
	"fmt"
	"strings"
)

// RegionalScope constructs a regional scope string from project ID and region.
func RegionalScope(projectID, region string) string {
	return fmt.Sprintf("%s.%s", projectID, region)
}

// ZonalScope constructs a zonal scope string from project ID and zone.
func ZonalScope(projectID, zone string) string {
	return fmt.Sprintf("%s.%s", projectID, zone)
}

// LastPathComponent extracts the last component from a GCP resource URL.
// If the input does not contain a "/", it returns the input itself.
// If the input is empty or only slashes, it returns an empty string.
func LastPathComponent(url string) string {
	if url == "" {
		return ""
	}
	parts := strings.Split(url, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}

// ExtractPathParam extracts the value following a given key from a GCP resource name.
// For example, for input="projects/my-proj/locations/global/keyRings/my-kr/cryptoKeys/my-key"
// and key="cryptoKeys", it will return "my-key".
func ExtractPathParam(key, input string) string {
	parts := strings.Split(input, "/")
	for i, part := range parts {
		if part == key && len(parts) > i+1 {
			return parts[i+1]
		}
	}
	return ""
}
