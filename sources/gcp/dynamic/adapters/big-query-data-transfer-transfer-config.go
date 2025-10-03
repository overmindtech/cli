package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// BigQuery Data Transfer transfer config adapter
// Manages scheduled queries and data transfer configurations for BigQuery
var _ = registerableAdapter{
	sdpType: gcpshared.BigQueryDataTransferTransferConfig,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/bigquery/docs/reference/datatransfer/rest/v1/projects.transferConfigs/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		// GET https://bigquerydatatransfer.googleapis.com/v1/projects/{projectId}/locations/{locationId}/transferConfigs/{transferConfigId}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs/%s"),
		// Reference: https://cloud.google.com/bigquery/docs/reference/datatransfer/rest/v1/projects.locations.transferConfigs/list
		// GET https://bigquerydatatransfer.googleapis.com/v1/projects/{projectId}/locations/{locationId}/transferConfigs
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs"),
		SearchDescription:   "Search for BigQuery Data Transfer transfer configs in a location. Use the format \"location\" or \"projects/project_id/locations/location/transferConfigs/transfer_config_id\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "transferConfigs"},
		IAMPermissions:      []string{"bigquery.transfers.get"},
		PredefinedRole:      "roles/bigquery.user",
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// state: https://cloud.google.com/bigquery/docs/reference/datatransfer/rest/v1/projects.locations.transferConfigs#TransferState
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"destinationDatasetId": {
			ToSDPItemType:    gcpshared.BigQueryDataset,
			Description:      "If the BigQuery Dataset is deleted or updated: The transfer config may fail to write data. If the transfer config is updated: The dataset remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"dataSourceId": {
			ToSDPItemType:    gcpshared.BigQueryDataTransferDataSource,
			Description:      "If the Data Source is deleted or updated: The transfer config may fail to function. If the transfer config is updated: The data source remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"notificationPubsubTopic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: Notifications may fail to be sent. If the transfer config is updated: The Pub/Sub topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"encryptionConfiguration.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_data_transfer_config",
		Description: "id => projects/{projectId}/locations/{location}/transferConfigs/{configId}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigquery_data_transfer_config.id",
			},
		},
	},
}.Register()
