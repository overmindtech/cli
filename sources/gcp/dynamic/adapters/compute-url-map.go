package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var computeBackendImpact = &gcpshared.Impact{ //nolint:unused
	ToSDPITemType: gcpshared.ComputeBackendService,
	Description:   "If the Backend Service is updated or deleted: The URL Map's routing behavior may change or break. If the URL Map changes: The backend service remains structurally unaffected.",
	BlastPropagation: &sdp.BlastPropagation{
		In:  true,
		Out: false,
	},
}

// URL Map (global, project-level) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/urlMaps/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/urlMaps/{urlMap}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/urlMaps
var computeUrlMapAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeUrlMap,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps",
		),
		// The list response key and path segment for URL maps.
		UniqueAttributeKeys: []string{"urlMaps"},
		IAMPermissions: []string{
			"compute.urlMaps.get",
			"compute.urlMaps.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"defaultService": computeBackendImpact,
		"defaultRouteAction.weightedBackendServices.backendService":                  computeBackendImpact,
		"defaultRouteAction.requestMirrorPolicy.backendService":                      computeBackendImpact,
		"pathMatchers.defaultService":                                                computeBackendImpact,
		"pathMatchers.pathRules.service":                                             computeBackendImpact,
		"pathMatchers.routeRules.service":                                            computeBackendImpact,
		"pathMatchers.defaultRouteAction.weightedBackendServices.backendService":     computeBackendImpact,
		"pathMatchers.defaultRouteAction.requestMirrorPolicy.backendService":         computeBackendImpact,
		"pathMatchers.pathRules.routeAction.weightedBackendServices.backendService":  computeBackendImpact,
		"pathMatchers.pathRules.routeAction.requestMirrorPolicy.backendService":      computeBackendImpact,
		"pathMatchers.routeRules.routeAction.weightedBackendServices.backendService": computeBackendImpact,
		"pathMatchers.routeRules.routeAction.requestMirrorPolicy.backendService":     computeBackendImpact,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_url_map",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_url_map.name",
			},
		},
	},
}.Register()
