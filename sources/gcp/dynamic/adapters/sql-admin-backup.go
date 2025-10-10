package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// SQL Admin Backup adapter for Cloud SQL backups
var _ = registerableAdapter{
	sdpType: gcpshared.SQLAdminBackup,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/GetBackup
		// GET https://sqladmin.googleapis.com/v1/{name=projects/*/backups/*}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/ListBackups
		// GET https://sqladmin.googleapis.com/v1/{parent=projects/*}/backups
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://sqladmin.googleapis.com/v1/projects/%s/backups"),
		UniqueAttributeKeys: []string{"backups"},
		// HEALTH: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups#sqlbackupstate
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/sql/docs/mysql/iam-permissions#permissions-gcloud
		IAMPermissions: []string{"cloudsql.backupRuns.get", "cloudsql.backupRuns.list"},
		PredefinedRole: "roles/cloudsql.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"instance": {
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The instance cannot recover from the backup.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"kmsKey":        gcpshared.CryptoKeyImpactInOnly,
		"kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
		"backupRun": {
			ToSDPItemType:    gcpshared.SQLAdminBackupRun,
			Description:      "They are tightly coupled with the SQL Admin Backup.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
