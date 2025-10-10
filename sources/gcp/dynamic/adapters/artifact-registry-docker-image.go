package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Artifact Registry Docker Image adapter for container images in Artifact Registry
var _ = registerableAdapter{
	sdpType: gcpshared.ArtifactRegistryDockerImage,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/get?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{name=projects/*/locations/*/repositories/*/dockerImages/*}
		// IAM permissions: artifactregistry.dockerImages.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages/%s"),
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/list?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{parent=projects/*/locations/*/repositories/*}/dockerImages
		// IAM permissions: artifactregistry.dockerImages.list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages"),
		SearchDescription:   "Search for Docker images in Artifact Registry. Use the format \"location|repository_id\" or \"projects/[project]/locations/[location]/repository/[repository_id]/dockerImages/[docker_image]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories", "dockerImages"},
		IAMPermissions:      []string{"artifactregistry.dockerimages.get", "artifactregistry.dockerimages.list"},
		PredefinedRole:      "roles/artifactregistry.reader",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// This is a link to its parent resource: ArtifactRegistryRepository
		// Linker will extract the repository name from the image name.
		"name": {
			ToSDPItemType:    gcpshared.ArtifactRegistryRepository,
			Description:      "If the Artifact Registry Repository is deleted or updated: The Docker Image may become invalid or inaccessible. If the Docker Image is updated: The repository remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/artifact_registry_docker_image",
		Description: "name => projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}}. We should use search to extract relevant fields.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_artifact_registry_docker_image.name",
			},
		},
	},
}.Register()
