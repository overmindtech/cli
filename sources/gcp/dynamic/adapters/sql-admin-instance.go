package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Cloud SQL Instance adapter
// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/instances/get
// GET:  https://sqladmin.googleapis.com/sql/v1/projects/{project}/instances/{instance}
// LIST: https://sqladmin.googleapis.com/sql/v1/projects/{project}/instances
var sqlAdminInstanceAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.SQLAdminInstance,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://sqladmin.googleapis.com/sql/v1/projects/%s/instances/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://sqladmin.googleapis.com/sql/v1/projects/%s/instances",
		),
		// Uniqueness within a project is determined by the instance name segment in the path.
		UniqueAttributeKeys: []string{"instances"},
		IAMPermissions: []string{
			"cloudsql.instances.get",
			"cloudsql.instances.list",
		},
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/instances#SqlInstanceState
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// VPC network used for private service connectivity.
		"settings.ipConfiguration.privateNetwork": gcpshared.ComputeNetworkImpactInOnly,
		// CMEK used to encrypt the primary data disk.
		"diskEncryptionConfiguration.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// CMEK used for automated backups (if configured).
		"settings.backupConfiguration.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// Cloud Storage bucket for SQL Server audit logs.
		"settings.sqlServerAuditConfig.bucket": {
			Description:      "If the Storage Bucket is deleted or updated: The Cloud SQL Instance may fail to write audit logs. If the Cloud SQL Instance is updated: The bucket remains unaffected.",
			ToSDPItemType:    gcpshared.StorageBucket,
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Name of the primary (master) instance this replica depends on.
		"masterInstanceName": {
			Description:      "If the master instance is deleted or updated: This replica may lose replication or become stale. If this replica is updated: The master remains unaffected.",
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Failover replica for high availability; changes in the failover target can impact this instance's HA posture.
		"failoverReplica.name": {
			Description:      "If the failover replica is deleted or updated: High availability for this instance may be reduced or fail. If this instance is updated: The failover replica remains unaffected.",
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Read replicas sourced from this primary instance. Changes to this instance can impact replicas, but replica changes typically do not impact the primary.
		"replicaNames": {
			Description:      "If this primary instance is deleted or materially updated: Its replicas may become unavailable or invalid. Changes on replicas generally do not impact the primary.",
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		// Added: All assigned IP addresses (public or private). Treated as tightly coupled network identifiers.
		"ipAddresses.ipAddress": gcpshared.IPImpactBothWays,
		"ipv6Address":           gcpshared.IPImpactBothWays,
		// Added: Service account used by the instance for operations.
		"serviceAccountEmailAddress": gcpshared.IAMServiceAccountImpactInOnly,
		// Added: DNS name representing the instance endpoint.
		"dnsName": {
			Description:      "Tightly coupled with the Cloud SQL Instance endpoint.",
			ToSDPItemType:    stdlib.NetworkDNS,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/sql_database_instance",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_sql_database_instance.name",
			},
		},
	},
}.Register()
