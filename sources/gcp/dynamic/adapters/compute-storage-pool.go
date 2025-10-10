package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Storage Pool adapter for storage pools
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeStoragePool,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/storagePools/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools/{storagePool}
		GetEndpointBaseURLFunc: gcpshared.ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools
		ListEndpointFunc:    gcpshared.ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools"),
		UniqueAttributeKeys: []string{"storagePools"},
		IAMPermissions:      []string{"compute.storagePools.get", "compute.storagePools.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
