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

// IsRegion checks if a string represents a GCP region.
// GCP regions typically follow the pattern "x-y" (e.g., "us-central1").
func IsRegion(s string) bool {
	parts := strings.Split(s, "-")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// IsZone checks if a string represents a GCP zone.
// GCP zones typically follow the pattern "x-y-z" (e.g., "us-central1-a").
func IsZone(s string) bool {
	parts := strings.Split(s, "-")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}
