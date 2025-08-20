package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var computeRouterAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeRouter,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers/{router}
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers
		ListEndpointFunc:    gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers"),
		UniqueAttributeKeys: []string{"routers"},
		IAMPermissions:      []string{"compute.routers.get", "compute.routers.list"},
	},
}
