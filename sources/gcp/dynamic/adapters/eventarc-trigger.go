package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Eventarc Trigger adapter (IN DEVELOPMENT)
// Reference: https://cloud.google.com/eventarc/docs/reference/rest/v1/projects.locations.triggers/get
// GET:  https://eventarc.googleapis.com/v1/projects/{project}/locations/{location}/triggers/{trigger}
// LIST: https://eventarc.googleapis.com/v1/projects/{project}/locations/{location}/triggers
var eventarcTriggerAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.EventarcTrigger,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers/%s",
		),
		// List requires only the location (region or global) besides project.
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers",
		),
		UniqueAttributeKeys: []string{"locations", "triggers"},
		IAMPermissions: []string{
			"eventarc.triggers.get",
			"eventarc.triggers.list",
		},
	},
	// No blast propagation yet. TODO: Evaluate targets (Cloud Run service, GKE, etc.) for links.
	blastPropagation: map[string]*gcpshared.Impact{},
}.Register()
