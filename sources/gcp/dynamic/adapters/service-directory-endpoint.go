package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Service Directory Endpoint adapter for Service Directory endpoints
var _ = registerableAdapter{
	sdpType: gcpshared.ServiceDirectoryEndpoint,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/get
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints/*
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithFourQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints/%s"),
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/list
		// IAM Perm: servicedirectory.endpoints.list
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints"),
		SearchDescription:   "Search for endpoints by \"location|namespace_id|service_id\" or \"projects/[project_id]/locations/[location]/namespaces/[namespace_id]/services/[service_id]/endpoints/[endpoint_id]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "namespaces", "services", "endpoints"},
		IAMPermissions:      []string{"servicedirectory.endpoints.get", "servicedirectory.endpoints.list"},
		PredefinedRole:      "roles/servicedirectory.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"name": {
			ToSDPItemType:    gcpshared.ServiceDirectoryService,
			Description:      "If the Service Directory Service is deleted or updated: The Endpoint may lose its association or fail to resolve names. If the Endpoint is updated: The service remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// An IPv4 or IPv6 address.
		"address": gcpshared.IPImpactBothWays,
		// The Google Compute Engine network (VPC) of the endpoint in the format projects/<project number>/locations/global/networks/*.
		"network": gcpshared.ComputeNetworkImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_directory_endpoint",
		Description: "id => projects/*/locations/*/namespaces/*/services/*/endpoints/*",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_service_directory_endpoint.id",
			},
		},
	},
}.Register()
