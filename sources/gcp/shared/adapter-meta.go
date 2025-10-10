package shared

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// Scope defines the scope of a GCP resource.
type Scope string

const (
	ScopeProject  Scope = "project"
	ScopeRegional Scope = "regional"
	ScopeZonal    Scope = "zonal"
)

type EndpointFunc func(query string) string

// AdapterMeta contains metadata for a GCP dynamic adapter.
type AdapterMeta struct {
	Scope                  Scope
	GetEndpointBaseURLFunc func(queryParts ...string) (EndpointFunc, error)
	ListEndpointFunc       func(queryParts ...string) (string, error)
	SearchEndpointFunc     func(queryParts ...string) (EndpointFunc, error)
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

// We have group of functions that are similar in nature, however they cannot simplified into a generic function because
// of the different number of query parts they accept.
// Also, we want to keep the explicit logic for now for the sake of human readability.

func ProjectLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and query
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
	}
}

func ProjectLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func ProjectLevelEndpointFuncWithThreeQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, and 3 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 3 && queryParts[0] != "" && queryParts[1] != "" && queryParts[2] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1], queryParts[2])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func ProjectLevelEndpointFuncWithFourQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 5 { // project ID, and 4 parts of the query
		panic(fmt.Sprintf("format string must contain 5 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 4 && queryParts[0] != "" && queryParts[1] != "" && queryParts[2] != "" && queryParts[3] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1], queryParts[2], queryParts[3])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
	}
}

func ZoneLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, zone, and query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

func RegionalLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, region, and query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func ZoneLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, zone, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

func RegionalLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, region, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func ProjectLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	if strings.Count(format, "%s") != 1 {
		panic(fmt.Sprintf("format string must contain 1 %%s placeholder: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return fmt.Sprintf(format, adapterInitParams[0]), nil
		}
		return "", fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
	}
}

func RegionLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and region
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1]), nil
		}
		return "", fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func ZoneLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and zone
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1]), nil
		}
		return "", fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

// SDPAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
// This map is populated during source initiation by individual adapter files.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{}
