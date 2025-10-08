package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var bigTableAdminClusterAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.BigTableAdminCluster,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*/clusters/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s"),
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*/clusters
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters"),
		UniqueAttributeKeys: []string{"instances", "clusters"},
		IAMPermissions:      []string{"bigtable.clusters.get", "bigtable.clusters.list"},
		PredefinedRole:      "roles/bigtable.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters#State
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Customer-managed encryption key protecting data in this cluster.
		"encryptionConfig.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// This is a backlink to instance.
		// Framework will extract the instance name and create the linked item query with GET
		// NOTE: We prioritize the backlink over a forward link to BigTableAdminBackup
		// because the backlink is more critical for understanding the cluster's dependencies.
		"name": {
			ToSDPItemType: gcpshared.BigTableAdminInstance,
			Description:   "If the BigTableAdmin Instance is deleted or updated: The Cluster may become invalid or inaccessible. If the Cluster is updated: The instance remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	// No Terraform mapping
}.Register()
