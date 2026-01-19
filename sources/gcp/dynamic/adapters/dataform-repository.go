package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Dataform Repository adapter for Dataform repositories
var _ = registerableAdapter{
	sdpType: gcpshared.DataformRepository,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/get
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories/*
		// IAM permissions: dataform.repositories.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories/%s"),
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/list
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories"),
		SearchDescription:   "Search for Dataform repositories in a location. Use the format \"location\" or \"projects/[project_id]/locations/[location]/repositories/[repository_name]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories"},
		IAMPermissions:      []string{"dataform.repositories.get", "dataform.repositories.list"},
		PredefinedRole:      "roles/dataform.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The name of the Secret Manager secret version to use as an authentication token for Git operations. Must be in the format projects/*/secrets/*/versions/*.
		"gitRemoteSettings.authenticationTokenSecretVersion": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The Dataform Repository may fail to authenticate with the Git remote. If the Dataform Repository is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The name of the Secret Manager secret version to use as a ssh private key for Git operations. Must be in the format projects/*/secrets/*/versions/*.
		"gitRemoteSettings.sshAuthenticationConfig.userPrivateKeySecretVersion": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The Dataform Repository may fail to authenticate with the Git remote. If the Dataform Repository is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Name of the Secret Manager secret version used to interpolate variables into the .npmrc file for package installation operations.
		"npmrcEnvironmentVariablesSecretVersion": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The Dataform Repository may fail to install npm packages. If the Dataform Repository is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The URL of the Git remote repository. Can be HTTPS (e.g., https://github.com/user/repo.git) or SSH (e.g., git@github.com:user/repo.git).
		"gitRemoteSettings.url": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the Git remote URL becomes inaccessible: The Dataform Repository may fail to sync with the remote. If the Dataform Repository is updated: The Git remote remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The service account to run workflow invocations under.
		"serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		// The reference to a KMS encryption key.
		// If provided, it will be used to encrypt user data in the repository and all child resources.
		// It is not possible to add or update the encryption key after the repository is created.
		// Example: projects/{kms_project}/locations/{location}/keyRings/{key_location}/cryptoKeys/{key}
		"kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// A data encryption state of a Git repository if this Repository is protected by a KMS key.
		"dataEncryptionState.kmsKeyVersionName": gcpshared.CryptoKeyVersionImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataform_repository",
		Description: "id => projects/{{project}}/locations/{{region}}/repositories/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataform_repository.id",
			},
		},
	},
}.Register()
