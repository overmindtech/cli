package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Resource Policy adapter for resource policies
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeResourcePolicy,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.RegionalLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/resourcePolicies/%s"),
		// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/list
		ListEndpointFunc:    gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/resourcePolicies"),
		UniqueAttributeKeys: []string{"resourcePolicies"},
		IAMPermissions:      []string{"compute.resourcePolicies.get", "compute.resourcePolicies.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Cloud Storage bucket storage location where snapshots created by this policy are stored.
		// The storageLocations field can contain bucket names, gs:// URIs, or region identifiers.
		// The manual adapter linker will handle extraction of bucket names from various formats.
		"snapshotSchedulePolicy.snapshotProperties.storageLocations": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The Resource Policy may fail to create snapshots. If the Resource Policy is updated: The Storage Bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
