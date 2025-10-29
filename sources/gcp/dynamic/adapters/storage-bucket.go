package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Storage Bucket adapter for Google Cloud Storage buckets
var _ = registerableAdapter{
	sdpType: gcpshared.StorageBucket,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/storage/docs/json_api/v1/buckets/get
		// GET https://storage.googleapis.com/storage/v1/b/{bucket}
		GetEndpointBaseURLFunc: func(queryParts ...string) (gcpshared.EndpointFunc, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("bucket name cannot be empty: %v", queryParts)
		},
		// Reference: https://cloud.google.com/storage/docs/json_api/v1/buckets/list
		// GET https://storage.googleapis.com/storage/v1/b?project={project}
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://storage.googleapis.com/storage/v1/b?project=%s"),
		UniqueAttributeKeys: []string{"b"},
		IAMPermissions:      []string{"storage.buckets.get", "storage.buckets.list"},
		PredefinedRole:      "roles/storage.bucketViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// A Cloud KMS key that will be used to encrypt objects written to this bucket if no encryption method is specified as part of the object write request.
		"encryption.defaultKmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// Name of the network.
		// Format: projects/PROJECT_ID/global/networks/NETWORK_NAME
		"ipFilter.vpcNetworkSources.network": gcpshared.ComputeNetworkImpactInOnly,
		// The destination bucket where the current bucket's logs should be placed.
		"logging.logBucket": {
			ToSDPItemType:    gcpshared.LoggingBucket,
			Description:      "If the Logging Bucket is deleted or updated: The Storage Bucket may fail to write logs. If the Storage Bucket is updated: The Logging Bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_storage_bucket.name",
			},
		},
	},
}.Register()
