package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Run Worker Pool:
// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.workerPools/get
// GET:  https://run.googleapis.com/v2/projects/{project}/locations/{location}/workerPools/{workerPool}
// LIST: https://run.googleapis.com/v2/projects/{project}/locations/{location}/workerPools
var runWorkerPoolAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.RunWorkerPool,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/workerPools/%s",
		),
		// The list endpoint requires the location only.
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/workerPools",
		),
		// location|workerPool
		UniqueAttributeKeys: []string{"locations", "workerPools"},
		IAMPermissions: []string{
			"run.workerPools.get",
			"run.workerPools.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{},
}.Register()
