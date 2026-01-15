package shared

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RecordExtractScopeFromURIError records an error from ExtractScopeFromURI to the span.
// This should be called whenever ExtractScopeFromURI returns an error to help with observability.
func RecordExtractScopeFromURIError(ctx context.Context, uri string, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err, trace.WithAttributes(
			attribute.String("ovm.gcp.extractScopeFromURI.uri", uri),
			attribute.String("ovm.gcp.extractScopeFromURI.error", err.Error()),
		))
	}
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

// ZoneToRegion converts a GCP zone to a region.
// The fully-qualified name for a zone is made up of <region>-<zone>.
// For example, the fully qualified name for zone a in region us-central1 is us-central1-a.
// https://cloud.google.com/compute/docs/regions-zones#identifying_a_region_or_zone
func ZoneToRegion(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) < 2 {
		return ""
	}

	return strings.Join(parts[:len(parts)-1], "-")
}

// ExtractScopeFromURI extracts the scope from a GCP resource URI.
// It supports various URL formats including full HTTPS URLs, full resource names,
// service destination formats, and bare paths.
//
// Examples:
//   - Zonal scope: "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/disks/my-disk" → "my-project.us-central1-a"
//   - Regional scope: "projects/my-project/regions/us-central1/subnetworks/my-subnet" → "my-project.us-central1"
//   - Project scope: "https://www.googleapis.com/compute/v1/projects/my-project/global/networks/my-network" → "my-project"
//
// The function determines scope based on the location specifiers found in the path:
//   - If zones/{zone} or locations/{zone-format} is found → zonal scope (project.zone)
//   - If regions/{region} or locations/{region-format} is found (and no zone) → regional scope (project.region)
//   - If global keyword is found, or only project is found → project scope (project)
//
// Returns an error if:
//   - The project ID cannot be determined
//   - Conflicting location specifiers are found (e.g., both zones and regions)
//   - The URI format is invalid
//
// If an error occurs, it is automatically recorded to the span from the context for observability.
func ExtractScopeFromURI(ctx context.Context, uri string) (string, error) {
	if uri == "" {
		err := fmt.Errorf("URI is empty")
		RecordExtractScopeFromURIError(ctx, uri, err)
		return "", err
	}

	// Extract the path portion from various URL formats
	path := extractPathFromURI(uri)

	// Extract project, region, zone, and location from the path
	projectID := ExtractPathParam("projects", path)
	zone := ExtractPathParam("zones", path)
	region := ExtractPathParam("regions", path)
	location := ExtractPathParam("locations", path)

	// Check for global keyword
	hasGlobal := strings.Contains(path, "/global/") || location == "global"

	// Handle special case: projects/_/buckets (project placeholder, cannot determine scope)
	if projectID == "_" {
		err := fmt.Errorf("cannot determine scope from URI with project placeholder: %s", uri)
		RecordExtractScopeFromURIError(ctx, uri, err)
		return "", err
	}

	// Validate project is present (unless it's the special _ case, already handled)
	if projectID == "" {
		err := fmt.Errorf("cannot determine scope: project ID not found in URI: %s", uri)
		RecordExtractScopeFromURIError(ctx, uri, err)
		return "", err
	}

	// Check for conflicting location specifiers
	if zone != "" && region != "" {
		err := fmt.Errorf("cannot determine scope: both zones and regions found in URI: %s", uri)
		RecordExtractScopeFromURIError(ctx, uri, err)
		return "", err
	}
	if zone != "" && location != "" {
		err := fmt.Errorf("cannot determine scope: both zones and locations found in URI: %s", uri)
		RecordExtractScopeFromURIError(ctx, uri, err)
		return "", err
	}

	// Determine scope based on location specifiers found
	// Priority: zone > region > project (global)

	// Zonal scope: zones/{zone} or locations/{zone-format}
	if zone != "" {
		return ZonalScope(projectID, zone), nil
	}
	if location != "" && location != "global" {
		// Check if location is zone-format using ZoneToRegion
		// If ZoneToRegion returns a non-empty region, the location is a zone
		if extractedRegion := ZoneToRegion(location); extractedRegion != "" {
			// Location is zone-format
			return ZonalScope(projectID, location), nil
		}
		// Location is region-format
		return RegionalScope(projectID, location), nil
	}

	// Regional scope: regions/{region}
	if region != "" {
		return RegionalScope(projectID, region), nil
	}

	// Project scope: global keyword or no location specifiers
	if hasGlobal || location == "global" {
		return projectID, nil
	}

	// Project scope: only project found, no location specifiers
	return projectID, nil
}

// extractPathFromURI extracts the resource path from various GCP URI formats.
// It handles:
//   - Full HTTPS URLs: https://www.googleapis.com/compute/v1/projects/...
//   - Service-specific HTTPS URLs: https://compute.googleapis.com/compute/v1/projects/...
//   - Full resource names: //compute.googleapis.com/projects/...
//   - Service destination formats: pubsub.googleapis.com/projects/...
//   - Bare paths: projects/...
func extractPathFromURI(uri string) string {
	// Remove query parameters and fragments
	if idx := strings.IndexAny(uri, "?#"); idx != -1 {
		uri = uri[:idx]
	}

	// Handle full resource names: //service.googleapis.com/path
	if strings.HasPrefix(uri, "//") {
		// Find the path after the domain
		parts := strings.SplitN(uri[2:], "/", 2)
		if len(parts) > 1 {
			return parts[1]
		}
		return ""
	}

	// Handle HTTPS/HTTP URLs: https://domain/path or http://domain/path
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// Remove protocol
		uri = uri[strings.Index(uri, "://")+3:]
		// Find the path after the domain
		parts := strings.SplitN(uri, "/", 2)
		if len(parts) > 1 {
			path := parts[1]
			// Strip version paths like /v1/, /v2/, /compute/v1/, /bigquery/v2/, etc.
			// These appear after the domain and before the resource path
			// Pattern: /{service}/v{version}/ or /v{version}/
			path = stripVersionPath(path)
			return path
		}
		return ""
	}

	// Handle service destination formats: service.googleapis.com/path
	// These don't have a protocol prefix
	if strings.Contains(uri, ".googleapis.com/") {
		parts := strings.SplitN(uri, ".googleapis.com/", 2)
		if len(parts) > 1 {
			path := parts[1]
			path = stripVersionPath(path)
			return path
		}
	}

	// Bare path: projects/... (use as-is)
	return uri
}

// stripVersionPath removes version paths from the beginning of a path.
// Examples:
//   - "v1/projects/..." → "projects/..."
//   - "compute/v1/projects/..." → "projects/..."
//   - "bigquery/v2/projects/..." → "projects/..."
func stripVersionPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return path
	}

	// Check for version pattern at the start
	// Pattern 1: /v{version}/ (e.g., v1, v2)
	if len(parts) > 0 && strings.HasPrefix(parts[0], "v") && len(parts[0]) == 2 {
		// Skip version part
		if len(parts) > 1 {
			return strings.Join(parts[1:], "/")
		}
		return ""
	}

	// Pattern 2: /{service}/v{version}/ (e.g., compute/v1, bigquery/v2)
	if len(parts) > 1 && strings.HasPrefix(parts[1], "v") && len(parts[1]) == 2 {
		// Skip service and version parts
		if len(parts) > 2 {
			return strings.Join(parts[2:], "/")
		}
		return ""
	}

	return path
}
