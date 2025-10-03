package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var artifactRegistryRepositoryAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ArtifactRegistryRepository,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories/get?rep_location=global
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeProject,
		// GET: https://artifactregistry.googleapis.com/v1/projects/*/locations/*/repositories/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s"),
		// LIST: https://artifactregistry.googleapis.com/v1/{parent=projects/*/locations/*}/repositories
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories"),
		UniqueAttributeKeys: []string{"locations", "repositories"},
		IAMPermissions:      []string{"artifactregistry.repositories.get", "artifactregistry.repositories.list"},
		PredefinedRole:      "roles/artifactregistry.reader",
		// HEALTH: Not currently exposed on the Repository resource (no status field providing operational state)
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/artifact_registry_repository#attributes-reference",
		Description: "The id is in the format `projects/{project}/locations/{location}/repositories/{repository_id}`. We will use SEARCH to find the repository by this ID.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_artifact_registry_repository.id",
			},
		},
	},
}.Register()
