package shared

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// LocationLevel defines at which level of the GCP hierarchy a resource is located.
type LocationLevel string

const (
	ProjectLevel  LocationLevel = "project"
	RegionalLevel LocationLevel = "regional"
	ZonalLevel    LocationLevel = "zonal"
)

// EndpointFunc is a function that generates an API endpoint URL given a query and location.
type EndpointFunc func(query string, location LocationInfo) string

// ListEndpointFunc is a function that generates a list endpoint URL for a given location.
type ListEndpointFunc func(location LocationInfo) (string, error)

// AdapterMeta contains metadata for a GCP dynamic adapter.
type AdapterMeta struct {
	LocationLevel LocationLevel
	// GetEndpointFunc is a function that generates GET endpoint URLs.
	// It receives the query string and LocationInfo and returns the URL.
	GetEndpointFunc EndpointFunc
	// ListEndpointFunc is a function that generates list endpoint URLs.
	// It accepts LocationInfo directly for the multi-scope architecture.
	ListEndpointFunc ListEndpointFunc
	// SearchEndpointFunc is a function that generates SEARCH endpoint URLs.
	// It receives the query string and LocationInfo and returns the URL.
	SearchEndpointFunc EndpointFunc
	// We will normally generate the search description from the UniqueAttributeKeys
	// but we allow it to be overridden for specific adapters.
	SearchDescription   string
	SDPAdapterCategory  sdp.AdapterCategory
	UniqueAttributeKeys []string
	InDevelopment       bool     // If true, the adapter is in development and should not be used in production.
	IAMPermissions      []string // List of IAM permissions required to access this resource.
	PredefinedRole      string   // Predefined role required to access this resource.
	NameSelector        string   // By default, it is `name`, but can be overridden for outlier cases
	// By default, we use the last item of the UniqueAttributeKeys.
	// However, there is an exception: https://cloud.google.com/dataproc/docs/reference/rest/v1/ListAutoscalingPoliciesResponse
	// Expected: `autoscalingPolicies` by convention, but the API returns `policies`
	ListResponseSelector string
}

// =============================================
// NEW PATTERN: Endpoint builder functions
// These take a format string and return an EndpointFunc
// =============================================

// ProjectLevelEndpointFuncWithSingleQuery returns a function that builds GET endpoint URLs for project-level resources.
// Format string should have 2 %s placeholders: project ID and query.
func ProjectLevelEndpointFuncWithSingleQuery(format string) EndpointFunc {
	if strings.Count(format, "%s") != 2 {
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, query)
	}
}

// ProjectLevelEndpointFuncWithTwoQueries returns a function for project-level resources with composite query.
// Format string should have 3 %s placeholders: project ID and 2 parts of the query.
func ProjectLevelEndpointFuncWithTwoQueries(format string) EndpointFunc {
	if strings.Count(format, "%s") != 3 {
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		queryParts := strings.Split(query, shared.QuerySeparator)
		if len(queryParts) != 2 || queryParts[0] == "" || queryParts[1] == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, queryParts[0], queryParts[1])
	}
}

// ProjectLevelEndpointFuncWithThreeQueries returns a function for project-level resources with 3-part query.
// Format string should have 4 %s placeholders: project ID and 3 parts of the query.
func ProjectLevelEndpointFuncWithThreeQueries(format string) EndpointFunc {
	if strings.Count(format, "%s") != 4 {
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		queryParts := strings.Split(query, shared.QuerySeparator)
		if len(queryParts) != 3 || queryParts[0] == "" || queryParts[1] == "" || queryParts[2] == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, queryParts[0], queryParts[1], queryParts[2])
	}
}

// ProjectLevelEndpointFuncWithFourQueries returns a function for project-level resources with 4-part query.
// Format string should have 5 %s placeholders: project ID and 4 parts of the query.
func ProjectLevelEndpointFuncWithFourQueries(format string) EndpointFunc {
	if strings.Count(format, "%s") != 5 {
		panic(fmt.Sprintf("format string must contain 5 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		queryParts := strings.Split(query, shared.QuerySeparator)
		if len(queryParts) != 4 || queryParts[0] == "" || queryParts[1] == "" || queryParts[2] == "" || queryParts[3] == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, queryParts[0], queryParts[1], queryParts[2], queryParts[3])
	}
}

// ZoneLevelEndpointFunc returns a function that builds GET endpoint URLs for zonal resources.
// Format string should have 3 %s placeholders: project ID, zone, and query.
func ZoneLevelEndpointFunc(format string) EndpointFunc {
	if strings.Count(format, "%s") != 3 {
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, location.Zone, query)
	}
}

// ZoneLevelEndpointFuncWithTwoQueries returns a function for zonal resources with composite query.
// Format string should have 4 %s placeholders: project ID, zone, and 2 parts of the query.
func ZoneLevelEndpointFuncWithTwoQueries(format string) EndpointFunc {
	if strings.Count(format, "%s") != 4 {
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		queryParts := strings.Split(query, shared.QuerySeparator)
		if len(queryParts) != 2 || queryParts[0] == "" || queryParts[1] == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, location.Zone, queryParts[0], queryParts[1])
	}
}

// RegionalLevelEndpointFunc returns a function that builds GET endpoint URLs for regional resources.
// Format string should have 3 %s placeholders: project ID, region, and query.
func RegionalLevelEndpointFunc(format string) EndpointFunc {
	if strings.Count(format, "%s") != 3 {
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, location.Region, query)
	}
}

// RegionalLevelEndpointFuncWithTwoQueries returns a function for regional resources with composite query.
// Format string should have 4 %s placeholders: project ID, region, and 2 parts of the query.
func RegionalLevelEndpointFuncWithTwoQueries(format string) EndpointFunc {
	if strings.Count(format, "%s") != 4 {
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(query string, location LocationInfo) string {
		if query == "" {
			return ""
		}
		queryParts := strings.Split(query, shared.QuerySeparator)
		if len(queryParts) != 2 || queryParts[0] == "" || queryParts[1] == "" {
			return ""
		}
		return fmt.Sprintf(format, location.ProjectID, location.Region, queryParts[0], queryParts[1])
	}
}

// =============================================
// LIST ENDPOINT FUNCTIONS
// =============================================

// ProjectLevelListFunc returns a ListEndpointFunc for project-level resources.
// Format string should have 1 %s placeholder: project ID.
func ProjectLevelListFunc(format string) ListEndpointFunc {
	if strings.Count(format, "%s") != 1 {
		panic(fmt.Sprintf("format string must contain 1 %%s placeholder: %s", format))
	}
	return func(location LocationInfo) (string, error) {
		if location.ProjectID == "" {
			return "", fmt.Errorf("project ID cannot be empty")
		}
		return fmt.Sprintf(format, location.ProjectID), nil
	}
}

// RegionLevelListFunc returns a ListEndpointFunc for regional resources.
// Format string should have 2 %s placeholders: project ID and region.
func RegionLevelListFunc(format string) ListEndpointFunc {
	if strings.Count(format, "%s") != 2 {
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(location LocationInfo) (string, error) {
		if location.ProjectID == "" || location.Region == "" {
			return "", fmt.Errorf("project ID and region cannot be empty")
		}
		return fmt.Sprintf(format, location.ProjectID, location.Region), nil
	}
}

// ZoneLevelListFunc returns a ListEndpointFunc for zonal resources.
// Format string should have 2 %s placeholders: project ID and zone.
func ZoneLevelListFunc(format string) ListEndpointFunc {
	if strings.Count(format, "%s") != 2 {
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(location LocationInfo) (string, error) {
		if location.ProjectID == "" || location.Zone == "" {
			return "", fmt.Errorf("project ID and zone cannot be empty")
		}
		return fmt.Sprintf(format, location.ProjectID, location.Zone), nil
	}
}

// SDPAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
// This map is populated during source initiation by individual adapter files.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{}
