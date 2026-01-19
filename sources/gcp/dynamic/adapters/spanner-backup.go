package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Spanner Backup adapter for Cloud Spanner backups
var _ = registerableAdapter{
	sdpType: gcpshared.SpannerBackup,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		InDevelopment:      true,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference:https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.backups/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*/backups/*
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://spanner.googleapis.com/v1/projects/%s/instances/%s/backups/%s"),
		// https://spanner.googleapis.com/v1/projects/*/instances/*/backups
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instances/%s/backups"),
		UniqueAttributeKeys: []string{"instances", "backups"},
		IAMPermissions:      []string{"spanner.backups.get", "spanner.backups.list"},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// This is a backlink to instance.
		// Framework will extract the instance name and create the linked item query with GET
		"name": {
			Description:      "If the Spanner Instance is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The instance remains unaffected.",
			ToSDPItemType:    gcpshared.SpannerInstance,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Name of the database from which this backup is created.
		"database": {
			Description:      "If the Spanner Database is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The database remains unaffected.",
			ToSDPItemType:    gcpshared.SpannerDatabase,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Names of databases restored from this backup. May be across instances.
		"referencingDatabases": {
			Description:      "If any of the databases restored from this backup are deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The restored databases remain unaffected.",
			ToSDPItemType:    gcpshared.SpannerDatabase,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Names of destination backups copying this source backup.
		"referencingBackups": {
			Description:      "If any of the destination backups copying this source backup are deleted or updated: The source backup may become invalid or inaccessible. If the source backup is updated: The destination backups remain unaffected.",
			ToSDPItemType:    gcpshared.SpannerBackup,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"encryptionInfo.kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
		// All Cloud KMS key versions used for encrypting the backup.
		"encryptionInformation.kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
		// URIs of backup schedules associated with this backup (only for scheduled backups).
		"backupSchedules": {
			Description:      "If any of the backup schedules associated with this backup are deleted or updated: The Backup may stop being created automatically. If the Backup is updated: The backup schedules remain unaffected.",
			ToSDPItemType:    gcpshared.SpannerBackupSchedule,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The instance partitions storing the backup (from the state at versionTime).
		"instancePartitions.instancePartition": {
			Description:      "If any of the instance partitions storing this backup are deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The instance partitions remain unaffected.",
			ToSDPItemType:    gcpshared.SpannerInstancePartition,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
