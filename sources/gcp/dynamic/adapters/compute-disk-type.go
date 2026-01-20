package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Disk Type adapter for persistent disk types
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeDiskType,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/diskTypes/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		LocationLevel:      gcpshared.ZonalLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes/{diskType}
		GetEndpointFunc: gcpshared.ZoneLevelEndpointFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/diskTypes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes
		ListEndpointFunc:    gcpshared.ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/diskTypes"),
		UniqueAttributeKeys: []string{"diskTypes"},
		IAMPermissions:      []string{"compute.diskTypes.get", "compute.diskTypes.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
