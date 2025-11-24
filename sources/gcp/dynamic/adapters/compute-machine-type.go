package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Machine Type adapter for machine type configurations
// Machine types define the hardware configuration (CPU, memory) available for compute instances in a zone.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/machineTypes
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/machineTypes/{machineType}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/machineTypes
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeMachineType,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/machineTypes/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/machineTypes/{machineType}
		GetEndpointBaseURLFunc: gcpshared.ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/machineTypes
		ListEndpointFunc:    gcpshared.ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes"),
		UniqueAttributeKeys: []string{"machineTypes"},
		IAMPermissions:      []string{"compute.machineTypes.get", "compute.machineTypes.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"accelerators.acceleratorType": {
			Description:      "If the Accelerator Type is deleted or deprecated: The machine type may no longer support that accelerator configuration. If the machine type is updated: The accelerator type remains unaffected.",
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
	},
}.Register()
