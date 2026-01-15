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
		PredefinedRole: "roles/dataplex.viewer",
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
			// Note: data.resource can reference either a Storage Bucket (for DataDiscoveryScan)
			// or a BigQuery Table (for DataProfileScan, DataQualityScan, or DataDocumentationScan).
			// The StorageBucket manual linker will detect BigQuery Table URIs and delegate to
			// the BigQueryTable linker automatically.
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the data source (Storage Bucket or BigQuery Table) is deleted or inaccessible: The data scan will fail to access the data source. If the data scan is updated: The data source remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Post-scan action BigQuery table exports
		"dataQualitySpec.postScanActions.bigqueryExport.resultsTable": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery table for exporting data quality scan results is deleted or inaccessible: The post-scan action will fail. If the data scan is updated: The BigQuery table remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"dataProfileSpec.postScanActions.bigqueryExport.resultsTable": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery table for exporting data profile scan results is deleted or inaccessible: The post-scan action will fail. If the data scan is updated: The BigQuery table remains unaffected.",
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
