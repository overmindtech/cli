package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Dataplex Entry Group adapter for Dataplex entry groups
var _ = registerableAdapter{
	sdpType: gcpshared.DataplexEntryGroup,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/get
		// GET https://dataplex.googleapis.com/v1/{name=projects/*/locations/*/entryGroups/*}
		// IAM permissions: dataplex.entryGroups.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups/%s"),
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/list
		// GET https://dataplex.googleapis.com/v1/{parent=projects/*/locations/*}/entryGroups
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups"),
		SearchDescription:   "Search for Dataplex entry groups in a location. Use the format \"location\" or \"projects/[project_id]/locations/[location]/entryGroups/[entry_group_id]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "entryGroups"},
		// HEALTH: https://cloud.google.com/dataplex/docs/reference/rest/v1/TransferStatus
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"dataplex.entryGroups.get", "dataplex.entryGroups.list"},
		PredefinedRole: "roles/dataplex.catalogViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no links for this item type.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataplex_entry_group#entry_group_id",
		Description: "id => projects/{{project}}/locations/{{location}}/entryGroups/{{entry_group_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataplex_entry_group.id",
			},
		},
	},
}.Register()
