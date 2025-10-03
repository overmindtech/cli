package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Target Pool (regional) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/targetPools/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/targetPools/{targetPool}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/targetPools
var computeTargetPoolAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeTargetPool,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools",
		),
		// Provide a no-op search for terraform mapping support with full resource ID.
		// Expected search query: projects/{project}/regions/{region}/targetPools/{name}
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 2 || adapterInitParams[0] == "" || adapterInitParams[1] == "" {
				return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
		},
		SearchDescription: "Search with full ID: projects/[project]/regions/[region]/targetPools/[name] (used for terraform mapping).",
		// The list response key for items is "targetPools".
		UniqueAttributeKeys: []string{"targetPools"},
		IAMPermissions: []string{
			"compute.targetPools.get",
			"compute.targetPools.list",
		},
		PredefinedRole: "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"instances": {
			ToSDPItemType:    gcpshared.ComputeInstance,
			Description:      "If the Compute Instance is deleted or updated: the pool membership becomes invalid or traffic may fail to reach it. If the pool is updated: the instance remains structurally unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"healthChecks": {
			ToSDPItemType:    gcpshared.ComputeHealthCheck,
			Description:      "If the Health Check is updated or deleted: health status and traffic distribution may be affected. If the pool is updated: the health check remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"backupPool": {
			ToSDPItemType:    gcpshared.ComputeTargetPool,
			Description:      "If the backup Target Pool is updated or deleted: failover behavior may change. If this pool is updated: the backup pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_target_pool",
		Description: "id => projects/{{project}}/regions/{{region}}/targetPools/{{name}}. We need to use SEARCH.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_compute_target_pool.id",
			},
		},
	},
}.Register()
