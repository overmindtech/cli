package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud File Instance adapter
// Cloud File provides managed NFS file servers for applications that require a filesystem interface and a shared filesystem for data.
//
// Adapter for GCP Cloud File Instance
// API Get:  https://cloud.google.com/filestore/docs/reference/rest/v1/projects.locations.instances/get
// API List: https://cloud.google.com/filestore/docs/reference/rest/v1/projects.locations.instances/list
var _ = registerableAdapter{
	sdpType: gcpshared.FileInstance,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		// Project-level adapter (uses locations path parameter)
		Scope: gcpshared.ScopeProject,
		// GET https://file.googleapis.com/v1/projects/{project}/locations/{location}/instances/{instance}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://file.googleapis.com/v1/projects/%s/locations/%s/instances/%s",
		),
		// Search (per-location) https://file.googleapis.com/v1/projects/{project}/locations/{location}/instances
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://file.googleapis.com/v1/projects/%s/locations/%s/instances",
		),
		SearchDescription:   "Search for Filestore instances in a location. Use the location string or the full resource name supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "instances"},
		IAMPermissions: []string{
			"file.instances.get",
			"file.instances.list",
		},
		PredefinedRole: "roles/file.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631 => state
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"networks.network":     gcpshared.ComputeNetworkImpactInOnly,
		"networks.ipAddresses": gcpshared.IPImpactBothWays,
		"kmsKeyName":           gcpshared.CryptoKeyImpactInOnly,

		"fileShares.sourceBackup": {
			ToSDPItemType:    gcpshared.FileBackup,
			Description:      "If the referenced Backup is deleted or updated: Restores or incremental backups may fail. If the File instance is updated: The backup remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/filestore_instance",
		Description: "id => projects/{{project}}/locations/{{location}}/instances/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_filestore_instance.id",
			},
		},
	},
}.Register()
