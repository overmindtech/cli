package shared

import (
	"fmt"
	"strings"
)

// ExtractRegion extracts the region from a given string.
// Expected format: "projects/{project}/regions/{region}"
// I.e., "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default"
func ExtractRegion(input string) string {
	parts := strings.Split(input, "/")
	for i, part := range parts {
		if part == "regions" && len(parts) >= i+1 {
			return parts[i+1]
		}
	}

	return "" // Return empty string if "regions" not found
}

// ExtractZone extracts the zone from a given string.
// Expected format: "projects/{project}/zones/{zone}"
// I.e., "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/disks/integration-test-instance"
func ExtractZone(input string) string {
	parts := strings.Split(input, "/")
	for i, part := range parts {
		if part == "zones" && len(parts) >= i+1 {
			return parts[i+1]
		}
	}

	return "" // Return empty string if "zones" not found
}

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
