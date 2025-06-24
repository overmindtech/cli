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
//
// For example, for input="projects/my-proj/locations/global/keyRings/my-kr/cryptoKeys/my-key"
// and keys=["keyRings", "cryptoKeys"], it will return ["my-kr", "my-key"].
// If a key is not found, it will not be included in the results.
//
// If it fails to extract any values, it returns an empty slice.
//
// If it's a single part and no results were found for the given key(s), it returns the input itself.
// input => "my-managed-dns-zone", keys => "managedZones", output => ["my-managed-dns-zone"]
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

// ExtractPathParamsWithCount extracts path parameters from a fully qualified GCP resource name.
// It returns the last `count` path parameters from the input string.
//
// For example, for input="projects/my-proj/locations/global/keyRings/my-kr/cryptoKeys/my-key"
// and count=2, it will return ["my-kr", "my-key"].
func ExtractPathParamsWithCount(input string, count int) []string {
	if count <= 0 || input == "" {
		return nil
	}

	parts := strings.Split(strings.Trim(input, "/"), "/")
	if len(parts) < 2*count {
		return nil
	}

	var result []string
	for i := count - 1; i >= 0; i-- {
		step := 1 + 2*i
		result = append(result, parts[len(parts)-step])
	}

	return result
}
