package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Service Directory Service adapter for Service Directory services
var _ = registerableAdapter{
	sdpType: gcpshared.ServiceDirectoryService,
	meta: gcpshared.AdapterMeta{
		InDevelopment: true,
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s"),
		// https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services
		// IAM Perm: servicedirectory.services.list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services"),
		UniqueAttributeKeys: []string{"locations", "namespaces", "services"},
		IAMPermissions:      []string{"servicedirectory.services.get", "servicedirectory.services.list"},
		PredefinedRole:      "roles/servicedirectory.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
