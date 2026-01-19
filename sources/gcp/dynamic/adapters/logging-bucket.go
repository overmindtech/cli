package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Logging Bucket adapter for Cloud Logging buckets
var _ = registerableAdapter{
	sdpType: gcpshared.LoggingBucket,
	meta: gcpshared.AdapterMeta{
		// global is a type of location.
		// location is generally a region.
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*
		// IAM permissions: logging.buckets.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets
		// IAM permissions: logging.buckets.list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets"),
		UniqueAttributeKeys: []string{"locations", "buckets"},
		// HEALTH: Supports Health status: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LifecycleState
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"logging.buckets.get", "logging.buckets.list"},
		PredefinedRole: "roles/logging.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"cmekSettings.kmsKeyName":        gcpshared.CryptoKeyImpactInOnly,
		"cmekSettings.kmsKeyVersionName": gcpshared.CryptoKeyVersionImpactInOnly,
		"cmekSettings.serviceAccountId":  gcpshared.IAMServiceAccountImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
