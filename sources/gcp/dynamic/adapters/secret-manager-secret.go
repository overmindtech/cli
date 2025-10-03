package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Secret Manager Secret adapter.
// GCP Refs:
//   - API (GET):  https://cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets/get
//     GET https://secretmanager.googleapis.com/v1/projects/{project}/secrets/{secret}
//   - LIST:       https://cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets/list
//     GET https://secretmanager.googleapis.com/v1/projects/{project}/secrets
//   - Type:       https://cloud.google.com/secret-manager/docs/reference/rest/v1/projects.secrets#Secret
//
// Scope: Project-level (no locations segment in the resource path).
var secretManagerSecretAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.SecretManagerSecret,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://secretmanager.googleapis.com/v1/projects/%s/secrets",
		),
		UniqueAttributeKeys: []string{"secrets"},
		IAMPermissions: []string{
			"secretmanager.secrets.get",
			"secretmanager.secrets.list",
		},
		PredefinedRole: "roles/secretmanager.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// CMEK used with Automatic replication
		"replication.automatic.customerManagedEncryption.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// CMEK used with User-managed replication per replica
		"replication.userManaged.replicas.customerManagedEncryption.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// Pub/Sub topic which Secret Manager will publish to when control plane events occur on this secret.
		"topics.name": {
			ToSDPItemType: gcpshared.PubSubTopic,
			Description:   "If the Pub/Sub Topic is deleted or its policy changes: Secret event notifications may fail. If the Secret changes: The topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret",
		Description: "Use the secret_id to GET the secret within the project.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_secret_manager_secret.secret_id",
			},
		},
	},
}.Register()
