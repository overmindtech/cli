package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// SQL Admin Backup Run adapter for Cloud SQL backup runs
var _ = registerableAdapter{
	sdpType: gcpshared.SQLAdminBackupRun,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/get
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns/{id}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/list
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns"),
		UniqueAttributeKeys: []string{"instances", "backupRuns"},
		// HEALTH: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns#sqlbackuprunstatus
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/sql/docs/mysql/iam-permissions#permissions-gcloud
		IAMPermissions: []string{"cloudsql.backupRuns.get", "cloudsql.backupRuns.list"},
		PredefinedRole: "roles/cloudsql.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"instance": {
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			Description:      "They are tightly coupled",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"diskEncryptionConfiguration.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// The Cloud KMS key version used to encrypt the backup.
		// Format: projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}
		"diskEncryptionStatus.kmsKeyVersionName": gcpshared.CryptoKeyVersionImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
