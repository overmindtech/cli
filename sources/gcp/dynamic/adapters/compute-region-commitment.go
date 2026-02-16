package adapters

import (
	"github.com/overmindtech/workspace/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var _ = registerableAdapter{
	sdpType: gcpshared.ComputeRegionCommitment,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		LocationLevel:      gcpshared.RegionalLevel,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/regionCommitments/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/commitments/{commitment}
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/commitments/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/regionCommitments/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/commitments
		ListEndpointFunc:    gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/commitments"),
		UniqueAttributeKeys: []string{"commitments"},
		IAMPermissions:      []string{"compute.commitments.get", "compute.commitments.list"},
		PredefinedRole:      "roles/compute.viewer",
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/regionCommitments#Status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	linkRules: map[string]*gcpshared.Impact{
		"reservations.name": {
			ToSDPItemType: gcpshared.ComputeReservation,
			Description:   "If the Region Commitment is deleted or updated: Reservations that reference this commitment may lose associated discounts or resource guarantees. If the Reservation is updated or deleted: The commitment remains unaffected.",
		},
		"licenseResource.license": {
			ToSDPItemType: gcpshared.ComputeLicense,
			Description:   "If the Region Commitment is deleted or updated: Licenses that reference this commitment won't be affected. If the License is updated or deleted: The commitment may lose associated discounts or resource guarantees.",
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_commitment",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_region_commitment.name",
			},
		},
	},
}.Register()
