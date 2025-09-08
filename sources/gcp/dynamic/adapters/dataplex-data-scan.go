package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Dataplex Data Scan allows you to perform data quality checks, profiling, and discovery within data assets in Dataplex
// GCP Ref (GET): https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.dataScans/get
// GCP Ref (Schema): https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.dataScans#DataScan
// GET  https://dataplex.googleapis.com/v1/projects/{project}/locations/{location}/dataScans/{dataScan}
// LIST https://dataplex.googleapis.com/v1/projects/{project}/locations/{location}/dataScans
var _ = registerableAdapter{
	sdpType: gcpshared.DataplexDataScan,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans",
		),
		SearchDescription:   "Search for Dataplex data scans in a location. Use the location name e.g., 'us-central1' or the format \"projects/[project_id]/locations/[location]/dataScans/[data_scan_id]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "dataScans"},
		IAMPermissions: []string{
			"dataplex.dataScans.get",
			"dataplex.dataScans.list",
		},
		// TODO: https://linear.app/overmind/issue/ENG-631 state
		// https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.dataScans#DataScan
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Data source references - can scan various data sources
		"data.entity": {
			ToSDPItemType: gcpshared.DataplexEntity,
			Description:   "If the Dataplex Entity is deleted: The data scan will fail to access the data source. If the data scan is updated: The dataplex entity remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"data.resource": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the storage resource is deleted or inaccessible: The data scan will fail to access the data source. If the data scan is updated: The storage resource remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataplex_datascan",
		Description: "id => projects/{{project}}/locations/{{location}}/dataScans/{{data_scan_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataplex_datascan.id",
			},
		},
	},
}.Register()
