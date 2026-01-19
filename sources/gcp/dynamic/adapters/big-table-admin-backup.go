package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// BigTable Admin Backup adapter for Cloud Bigtable backups
var _ = registerableAdapter{
	sdpType: gcpshared.BigTableAdminBackup,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/clusters/*/backups/*}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s"),
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*/clusters/*}/backups
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups"),
		UniqueAttributeKeys: []string{"instances", "clusters", "backups"},
		// HEALTH: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"bigtable.backups.get", "bigtable.backups.list"},
		PredefinedRole: "roles/bigtable.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"name": {
			ToSDPItemType:    gcpshared.BigTableAdminCluster,
			Description:      "If the BigTableAdmin Cluster is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The cluster remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"sourceTable": {
			ToSDPItemType:    gcpshared.BigTableAdminTable,
			Description:      "If the BigTableAdmin Table is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The table remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"sourceBackup": {
			ToSDPItemType:    gcpshared.BigTableAdminBackup,
			Description:      "If the source Backup is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The source backup remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"encryptionInfo.kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
