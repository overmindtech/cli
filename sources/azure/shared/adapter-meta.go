package shared

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// LocationLevel defines the scope of an Azure resource.
type LocationLevel string

const (
	SubscriptionLevel  LocationLevel = "subscription"
	ResourceGroupLevel LocationLevel = "resource-group"
	RegionalLevel      LocationLevel = "regional"
)

type EndpointFunc func(query string) string

// AdapterMeta contains metadata for an Azure dynamic adapter.
type AdapterMeta struct {
	LocationLevel LocationLevel
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
	// Can be overridden for specific adapters if the API response structure differs.
	ListResponseSelector string
}

// We have group of functions that are similar in nature, however they cannot simplified into a generic function because
// of the different number of query parts they accept.
// Also, we want to keep the explicit logic for now for the sake of human readability.

// TODO: fix subscription-level endpoint functions to use subscriptionID instead of projectID in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
func SubscriptionLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // subscription ID and query
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
		return nil, fmt.Errorf("subscriptionID cannot be empty: %v", adapterInitParams)
	}
}

// TODO: fix subscription-level endpoint functions to use subscriptionID and resourceGroup instead of projectID in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
func ResourceGroupLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // subscription ID, resource group, and query
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
		return nil, fmt.Errorf("subscriptionID and resourceGroup cannot be empty: %v", adapterInitParams)
	}
}

// TODO: fix remaining endpoint functions (ProjectLevel, ZoneLevel, RegionalLevel) to use Azure scopes (subscription, resourceGroup) instead of GCP scopes (project, zone) in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
// These functions are currently GCP-specific and need to be refactored for Azure resource scoping

// SDPAssetTypeToAdapterMeta maps Azure asset types to their corresponding adapter metadata.
// This map is populated during source initiation by individual adapter files.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{}
