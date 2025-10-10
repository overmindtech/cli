package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Logging Link adapter for Cloud Logging links
var _ = registerableAdapter{
	sdpType: gcpshared.LoggingLink,
	meta: gcpshared.AdapterMeta{
		// HEALTH: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LifecycleState
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links/*
		// IAM permissions: logging.links.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links
		// IAM permissions: logging.links.list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links"),
		UniqueAttributeKeys: []string{"locations", "buckets", "links"},
		IAMPermissions:      []string{"logging.links.get", "logging.links.list"},
		PredefinedRole:      "roles/logging.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"name": {
			ToSDPItemType:    gcpshared.LoggingBucket,
			Description:      "If the Logging Bucket is deleted or updated: The Logging Link may lose its association or fail to function as expected. If the Logging Link is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"bigqueryDataset.datasetId": {
			Description:      "They are tightly coupled with the Logging Link.",
			ToSDPItemType:    gcpshared.BigQueryDataset,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
