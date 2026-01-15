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
	blastPropagation: map[string]*gcpshared.Impact{
		// Link from parent Service to child Endpoints via SEARCH
		// The framework will extract location, namespace, and service from the service name
		// and create a SEARCH query to find all endpoints under this service
		"name": {
			ToSDPItemType: gcpshared.ServiceDirectoryEndpoint,
			Description:   "If the Service Directory Service is deleted or updated: All associated endpoints may become invalid or inaccessible. If an endpoint is updated: The service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  false,
				Out: true,
			},
			IsParentToChild: true,
		},
		// Link to IP addresses in endpoint addresses (if endpoints are included in the response)
		// The linker will automatically detect if the value is an IP address or DNS name
		"endpoints.address": gcpshared.IPImpactBothWays,
		// Link to VPC networks referenced by endpoints
		"endpoints.network": gcpshared.ComputeNetworkImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
