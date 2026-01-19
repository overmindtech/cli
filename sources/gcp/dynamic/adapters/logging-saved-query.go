package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Logging Saved Query adapter for Cloud Logging saved queries
var _ = registerableAdapter{
	sdpType: gcpshared.LoggingSavedQuery,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries/*
		// IAM permissions: logging.savedQueries.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries
		// IAM permissions: logging.savedQueries.list
		// Saved Query has to be shared with the project (opposite is a private one) to show up here.
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries"),
		UniqueAttributeKeys: []string{"locations", "savedQueries"},
		// Documents lists `get` and `list` as the required iam permissions, but there is no permissions like that.
		// So, the closest one is chosen.
		// https://cloud.google.com/iam/docs/roles-permissions/logging
		IAMPermissions: []string{"logging.queries.getShared", "logging.queries.listShared"},
		PredefinedRole: "roles/logging.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no links for this item type.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
