package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Run Service adapter (IN DEVELOPMENT)
// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/get
// GET:  https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}
// LIST: https://run.googleapis.com/v2/projects/{project}/locations/{location}/services
var runServiceAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.RunService,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s",
		),
		// List requires only location in addition to project
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/services",
		),
		UniqueAttributeKeys: []string{"locations", "services"},
		IAMPermissions: []string{
			"run.services.get",
			"run.services.list",
		},
	},
	// No blast propagation defined yet. TODO: Evaluate references (e.g. revisions) if needed.
	blastPropagation: map[string]*gcpshared.Impact{},
}.Register()
