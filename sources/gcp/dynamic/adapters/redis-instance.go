package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// GCP Cloud Memorystore Redis Instance adapter.
// Cloud Memorystore for Redis provides a fully managed Redis service that is highly available and scalable.
// GCP Ref:
//   - API Call structure (GET): https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances/get
//     GET https://redis.googleapis.com/v1/projects/{project}/locations/{location}/instances/{instance}
//   - Type Definition (Instance): https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances#Instance
//   - LIST: https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances/list
//
// Scope: Project-level (uses locations path parameter; unique attributes include location+instance).
var redisInstanceAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.RedisInstance,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances/get
		// GET https://redis.googleapis.com/v1/projects/{project}/locations/{location}/instances/{instance}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://redis.googleapis.com/v1/projects/%s/locations/%s/instances/%s",
		),
		// Reference: https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances/list
		// GET https://redis.googleapis.com/v1/projects/{project}/locations/{location}/instances
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://redis.googleapis.com/v1/projects/%s/locations/%s/instances",
		),
		SearchDescription:   "Search Redis instances in a location. Use the format \"location\" or \"projects/[project_id]/locations/[location]/instances/[instance_name]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "instances"},
		IAMPermissions: []string{
			"redis.instances.get",
			"redis.instances.list",
		},
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations.instances#Instance.State
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The name of the VPC network to which the instance is connected.
		"authorizedNetwork": gcpshared.ComputeNetworkImpactInOnly,
		// Optional. The KMS key reference that the customer provides when trying to create the instance.
		"customerManagedKey": gcpshared.CryptoKeyImpactInOnly,
		// Output only. Hostname or IP address of the exposed Redis endpoint used by clients to connect to the service.
		"host": gcpshared.IPImpactBothWays,
		// Output only. List of server CA certificates for the instance.
		"serverCaCerts.cert": {
			ToSDPItemType:    gcpshared.ComputeSSLCertificate,
			Description:      "If the certificate is deleted or updated: The Redis instance may lose secure connectivity. If the Redis instance is updated: The certificate remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/redis_instance",
		Description: "id => projects/{project}/locations/{location}/instances/{instance}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_redis_instance.id",
			},
		},
	},
}.Register()
