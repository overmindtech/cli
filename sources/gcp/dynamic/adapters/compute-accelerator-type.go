package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Accelerator Type adapter for GPU/TPU accelerator types
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeAcceleratorType,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/acceleratorTypes/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		LocationLevel:      gcpshared.ZonalLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/acceleratorTypes/{acceleratorType}
		GetEndpointFunc: gcpshared.ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/acceleratorTypes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/acceleratorTypes
		ListEndpointFunc:    gcpshared.ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/acceleratorTypes"),
		UniqueAttributeKeys: []string{"acceleratorTypes"},
		IAMPermissions:      []string{"compute.acceleratorTypes.get", "compute.acceleratorTypes.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
