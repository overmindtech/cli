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
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/GetBackup
		// GET https://sqladmin.googleapis.com/v1/{name=projects/*/backups/*}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s"),
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
			ToSDPItemType: gcpshared.SQLAdminInstance,
			Description:   "If the Cloud SQL Instance is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The instance cannot recover from the backup.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"kmsKey":        gcpshared.CryptoKeyImpactInOnly,
		"kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
		// VPC network used for private IP access (from instance settings snapshot at backup time).
		"instanceSettings.settings.ipConfiguration.privateNetwork": gcpshared.ComputeNetworkImpactInOnly,
		// Allowed external IPv4 networks/ranges that can connect to the instance using its public IP (from instance settings snapshot).
		// Each entry uses CIDR notation (e.g., 203.0.113.0/24, 198.51.100.5/32).
		"instanceSettings.settings.ipConfiguration.authorizedNetworks.value": gcpshared.IPImpactBothWays,
		// Named allocated IP range for use (Private IP only, from instance settings snapshot).
		// This references an Internal Range resource that was used at backup time.
		"instanceSettings.settings.ipConfiguration.allocatedIpRange": {
			ToSDPItemType: gcpshared.NetworkConnectivityInternalRange,
			Description:   "If the Reserved Internal Range is deleted or updated: The backup's instance settings snapshot may reference an invalid IP range configuration. If the backup is updated: The internal range remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
