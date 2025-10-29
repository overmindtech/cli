package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Spanner Database adapter for Cloud Spanner databases
var _ = registerableAdapter{
	sdpType: gcpshared.SpannerDatabase,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*/databases/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases/%s"),
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases/list?rep_location=global
		// https://spanner.googleapis.com/v1/{parent=projects/*/instances/*}/databases
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases"),
		UniqueAttributeKeys: []string{"instances", "databases"},
		// HEALTH: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"spanner.databases.get", "spanner.databases.list"},
		PredefinedRole: "overmind_custom_role",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The Cloud KMS key used to encrypt the database.
		"encryptionConfig.kmsKeyName":  gcpshared.CryptoKeyImpactInOnly,
		"encryptionConfig.kmsKeyNames": gcpshared.CryptoKeyImpactInOnly,
		"restoreInfo.backupInfo.backup": {
			Description:      "If the Spanner Backup is deleted or updated: The Database may become invalid or inaccessible. If the Database is updated: The backup remains unaffected.",
			ToSDPItemType:    gcpshared.SpannerBackup,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"encryptionInfo.kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
		// This is a backlink to instance.
		// Framework will extract the instance name and create the linked item query with GET
		"name": {
			Description:      "If the Spanner Instance is deleted or updated: The Database may become invalid or inaccessible. If the Database is updated: The instance remains unaffected.",
			ToSDPItemType:    gcpshared.SpannerInstance,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/spanner_database.html",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_spanner_database.name",
			},
		},
	},
}.Register()
