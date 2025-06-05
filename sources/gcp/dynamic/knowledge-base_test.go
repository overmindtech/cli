package dynamic

import (
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSdpAssetTypeToAdapterMeta_Endpoints(t *testing.T) {
	type endpointTest struct {
		name        string
		assetType   shared.ItemType
		fnType      string // "get", "list", or "search"
		params      []string
		expectedURL string
		expectErr   bool
	}

	tests := []endpointTest{
		{
			name:        "ComputeNetwork GetEndpointBaseURLFunc",
			assetType:   gcpshared.ComputeNetwork,
			fnType:      "get",
			params:      []string{"my-proj"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/my-proj/global/networks/",
			expectErr:   false,
		},
		{
			name:        "ComputeNetwork ListEndpointFunc",
			assetType:   gcpshared.ComputeNetwork,
			fnType:      "list",
			params:      []string{"my-proj"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/my-proj/global/networks",
			expectErr:   false,
		},
		{
			name:        "ComputeSubnetwork GetEndpointBaseURLFunc",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "get",
			params:      []string{"my-proj", "us-central1"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/my-proj/regions/us-central1/subnetworks/",
			expectErr:   false,
		},
		{
			name:        "ComputeSubnetwork ListEndpointFunc",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "list",
			params:      []string{"my-proj", "us-central1"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/my-proj/regions/us-central1/subnetworks",
			expectErr:   false,
		},
		{
			name:        "BigQueryTable GetEndpointBaseURLFunc",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "get",
			params:      []string{"my-proj", "my-dataset"},
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/my-proj/datasets/my-dataset/tables/",
			expectErr:   false,
		},
		{
			name:        "BigQueryTable SearchEndpointFunc",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "search",
			params:      []string{"my-proj", "my-dataset"},
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/my-proj/datasets/my-dataset/tables",
			expectErr:   false,
		},
		{
			name:        "BigQueryDataset GetEndpointBaseURLFunc",
			assetType:   gcpshared.BigQueryDataset,
			fnType:      "get",
			params:      []string{"my-proj"},
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/my-proj/datasets/",
			expectErr:   false,
		},
		{
			name:        "BigQueryDataset ListEndpointFunc",
			assetType:   gcpshared.BigQueryDataset,
			fnType:      "list",
			params:      []string{"my-proj"},
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/my-proj/datasets",
			expectErr:   false,
		},
		{
			name:        "PubSubSubscription GetEndpointBaseURLFunc",
			assetType:   gcpshared.PubSubSubscription,
			fnType:      "get",
			params:      []string{"my-proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/my-proj/subscriptions/",
			expectErr:   false,
		},
		{
			name:        "PubSubSubscription ListEndpointFunc",
			assetType:   gcpshared.PubSubSubscription,
			fnType:      "list",
			params:      []string{"my-proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/my-proj/subscriptions",
			expectErr:   false,
		},
		{
			name:        "PubSubTopic GetEndpointBaseURLFunc",
			assetType:   gcpshared.PubSubTopic,
			fnType:      "get",
			params:      []string{"my-proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/my-proj/topics/",
			expectErr:   false,
		},
		{
			name:        "PubSubTopic ListEndpointFunc",
			assetType:   gcpshared.PubSubTopic,
			fnType:      "list",
			params:      []string{"my-proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/my-proj/topics",
			expectErr:   false,
		},
		// Error cases for missing required parameters
		{
			name:        "ComputeNetwork ListEndpointFunc missing projectID",
			assetType:   gcpshared.ComputeNetwork,
			fnType:      "list",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "ComputeSubnetwork GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "get",
			params:      []string{"", "us-central1"},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "ComputeSubnetwork GetEndpointBaseURLFunc missing region",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "get",
			params:      []string{"my-proj", ""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "ComputeSubnetwork ListEndpointFunc missing projectID",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "list",
			params:      []string{"", "us-central1"},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "ComputeSubnetwork ListEndpointFunc missing region",
			assetType:   gcpshared.ComputeSubnetwork,
			fnType:      "list",
			params:      []string{"my-proj", ""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryTable GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "get",
			params:      []string{"", "my-dataset"},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryTable GetEndpointBaseURLFunc missing datasetId",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "get",
			params:      []string{"my-proj", ""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryTable SearchEndpointFunc missing projectID",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "search",
			params:      []string{"", "my-dataset"},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryTable SearchEndpointFunc missing datasetId",
			assetType:   gcpshared.BigQueryTable,
			fnType:      "search",
			params:      []string{"my-proj", ""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryDataset GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.BigQueryDataset,
			fnType:      "get",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "BigQueryDataset ListEndpointFunc missing projectID",
			assetType:   gcpshared.BigQueryDataset,
			fnType:      "list",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "PubSubSubscription GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.PubSubSubscription,
			fnType:      "get",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "PubSubSubscription ListEndpointFunc missing projectID",
			assetType:   gcpshared.PubSubSubscription,
			fnType:      "list",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "PubSubTopic GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.PubSubTopic,
			fnType:      "get",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "PubSubTopic ListEndpointFunc missing projectID",
			assetType:   gcpshared.PubSubTopic,
			fnType:      "list",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
		{
			name:        "ComputeNetwork GetEndpointBaseURLFunc missing projectID",
			assetType:   gcpshared.ComputeNetwork,
			fnType:      "get",
			params:      []string{""},
			expectedURL: "",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := sdpAssetTypeToAdapterMeta[tt.assetType]
			if !ok {
				t.Fatalf("assetType %v not found in sdpAssetTypeToAdapterMeta", tt.assetType)
			}

			var (
				gotURL string
				err    error
			)
			switch tt.fnType {
			case "get":
				gotURL, err = meta.GetEndpointBaseURLFunc(tt.params...)
			case "list":
				if meta.ListEndpointFunc == nil {
					t.Skip("ListEndpointFunc not defined for this asset type")
				}
				gotURL, err = meta.ListEndpointFunc(tt.params...)
			case "search":
				if meta.SearchEndpointFunc == nil {
					t.Skip("SearchEndpointFunc not defined for this asset type")
				}
				gotURL, err = meta.SearchEndpointFunc(tt.params...)
			default:
				t.Fatalf("unknown fnType: %s", tt.fnType)
			}

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)", tt.params)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v (params: %v)", err, tt.params)
				}
				if gotURL != tt.expectedURL {
					t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", gotURL, tt.expectedURL)
				}
			}
		})
	}
}
