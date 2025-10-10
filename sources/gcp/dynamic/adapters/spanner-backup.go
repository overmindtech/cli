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
		Scope:              gcpshared.ScopeProject,
		// Reference:https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.backups/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*/backups/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://spanner.googleapis.com/v1/projects/%s/instances/%s/backups/%s"),
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
		"encryptionInfo.kmsKeyVersion": gcpshared.CryptoKeyVersionImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
