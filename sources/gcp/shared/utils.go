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

// ExtractPathParams extracts values following specified keys from a GCP resource name.
// It returns a slice of values in the order of the keys provided.
// For example, for input="projects/my-proj/locations/global/keyRings/my-kr/cryptoKeys/my-key"
// and keys=["keyRings", "cryptoKeys"], it will return ["my-kr", "my-key"].
// If a key is not found, it will not be included in the results.
// If it fails to extract any values, it returns an empty slice.
func ExtractPathParams(input string, keys ...string) []string {
	parts := strings.Split(input, "/")
	results := make([]string, 0, len(keys))

	for k := 0; k <= len(keys)-1; k++ {
		key := keys[k]
		for i, part := range parts {
			if part == key && len(parts) > i+1 {
				results = append(results, parts[i+1])
				break
			}
		}
	}

	// if it's a single part and no results were found, return the part itself
	if len(results) == 0 && len(parts) == 1 && parts[0] != "" {
		return []string{parts[0]}
	}

	return results
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

// LastSegmentsCount is the number of segments to keep when shortening a self link
const LastSegmentsCount = 4

// ShortenSelfLink shortens a given link for human readability
// https://www.googleapis.com/compute/v1/projects/test-457614/zones/us-central1-c/instanceGroupManagers/overmind-integration-test-igm-default
// /zones/us-central1-c/instanceGroupManagers/overmind-integration-test-igm-default
// this is a primitive initial work
func ShortenSelfLink(selfLink string) string {
	parts := strings.Split(selfLink, "/")

	if len(parts) < LastSegmentsCount {
		return selfLink
	}

	return strings.Join(parts[len(parts)-LastSegmentsCount:], "/")
}
