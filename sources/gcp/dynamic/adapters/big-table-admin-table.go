package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// BigTable Admin Table adapter for Cloud Bigtable tables
var _ = registerableAdapter{
	sdpType: gcpshared.BigTableAdminTable,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/tables/*}
		// IAM permissions: bigtable.tables.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/list
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*}/tables
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables"),
		SearchDescription:   "Search for BigTable tables in an instance. Use the format \"instance_name\" or \"projects/[project_id]/instances/[instance_name]/tables/[table_name]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "tables"},
		IAMPermissions:      []string{"bigtable.tables.get", "bigtable.tables.list"},
		PredefinedRole:      "roles/bigtable.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"name": {
			ToSDPItemType:    gcpshared.BigTableAdminInstance,
			Description:      "If the BigTableAdmin Instance is deleted or updated: The Table may become invalid or inaccessible. If the Table is updated: The instance remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// If this table was restored from another data source (e.g. a backup), this field, restoreInfo, will be populated with information about the restore.
		"restoreInfo.backupInfo.sourceTable": {
			ToSDPItemType:    gcpshared.BigTableAdminTable,
			Description:      "If the source BigTableAdmin Table is deleted or updated: The restored table may become invalid or inaccessible. If the restored table is updated: The source table remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"restoreInfo.backupInfo.sourceBackup": {
			ToSDPItemType:    gcpshared.BigTableAdminBackup,
			Description:      "If the source BigTableAdmin Backup is deleted or updated: The restored table may become invalid or inaccessible. If the restored table is updated: The source backup remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_table",
		Description: "id => projects/{{project}}/instances/{{instance_name}}/tables/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_table.id",
			},
		},
	},
}.Register()
