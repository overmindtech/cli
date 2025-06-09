package shared

import (
	"testing"

	"github.com/overmindtech/cli/sources/shared"
)

func TestSDPAssetTypeToAdapterMeta_GetEndpointBaseURLFunc(t *testing.T) {
	tests := []struct {
		name        string
		assetType   shared.ItemType
		params      []string
		query       string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "ComputeNetwork valid",
			assetType:   ComputeNetwork,
			params:      []string{"proj"},
			query:       "net",
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/global/networks/net",
			expectErr:   false,
		},
		{
			name:      "ComputeNetwork missing param",
			assetType: ComputeNetwork,
			params:    []string{""},
			query:     "net",
			expectErr: true,
		},
		{
			name:        "ComputeSubnetwork valid",
			assetType:   ComputeSubnetwork,
			params:      []string{"proj", "region"},
			query:       "subnet",
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/regions/region/subnetworks/subnet",
			expectErr:   false,
		},
		{
			name:      "ComputeSubnetwork missing region",
			assetType: ComputeSubnetwork,
			params:    []string{"proj", ""},
			query:     "subnet",
			expectErr: true,
		},
		{
			name:        "BigQueryDataset valid",
			assetType:   BigQueryDataset,
			params:      []string{"proj"},
			query:       "dataset",
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/proj/datasets/dataset",
			expectErr:   false,
		},
		{
			name:      "BigQueryDataset missing param",
			assetType: BigQueryDataset,
			params:    []string{""},
			query:     "dataset",
			expectErr: true,
		},
		{
			name:        "PubSubSubscription valid",
			assetType:   PubSubSubscription,
			params:      []string{"proj"},
			query:       "mysub",
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/subscriptions/mysub",
			expectErr:   false,
		},
		{
			name:      "PubSubSubscription missing param",
			assetType: PubSubSubscription,
			params:    []string{""},
			query:     "mysub",
			expectErr: true,
		},
		{
			name:        "PubSubTopic valid",
			assetType:   PubSubTopic,
			params:      []string{"proj"},
			query:       "mytopic",
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/topics/mytopic",
			expectErr:   false,
		},
		{
			name:      "PubSubTopic missing param",
			assetType: PubSubTopic,
			params:    []string{""},
			query:     "mytopic",
			expectErr: true,
		},
		{
			name:        "ComputeInstance valid",
			assetType:   ComputeInstance,
			params:      []string{"proj", "zone"},
			query:       "inst",
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/zones/zone/instances/inst",
			expectErr:   false,
		},
		{
			name:      "ComputeInstance missing zone",
			assetType: ComputeInstance,
			params:    []string{"proj", ""},
			query:     "inst",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := SDPAssetTypeToAdapterMeta[tt.assetType]
			if !ok {
				t.Fatalf("assetType %v not found in SDPAssetTypeToAdapterMeta", tt.assetType)
			}
			urlFunc, err := meta.GetEndpointBaseURLFunc(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, urlFunc)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, urlFunc, tt.expectedURL)
				return
			}
			gotURL := urlFunc(tt.query)
			if gotURL != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", gotURL, tt.expectedURL)
			}
		})
	}
}

func TestSDPAssetTypeToAdapterMeta_ListEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		assetType   shared.ItemType
		params      []string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "ComputeNetwork valid",
			assetType:   ComputeNetwork,
			params:      []string{"proj"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/global/networks",
			expectErr:   false,
		},
		{
			name:      "ComputeNetwork missing param",
			assetType: ComputeNetwork,
			params:    []string{""},
			expectErr: true,
		},
		{
			name:        "ComputeSubnetwork valid",
			assetType:   ComputeSubnetwork,
			params:      []string{"proj", "region"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/regions/region/subnetworks",
			expectErr:   false,
		},
		{
			name:      "ComputeSubnetwork missing region",
			assetType: ComputeSubnetwork,
			params:    []string{"proj", ""},
			expectErr: true,
		},
		{
			name:        "BigQueryDataset valid",
			assetType:   BigQueryDataset,
			params:      []string{"proj"},
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/proj/datasets",
			expectErr:   false,
		},
		{
			name:      "BigQueryDataset missing param",
			assetType: BigQueryDataset,
			params:    []string{""},
			expectErr: true,
		},
		{
			name:        "PubSubSubscription valid",
			assetType:   PubSubSubscription,
			params:      []string{"proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/subscriptions",
			expectErr:   false,
		},
		{
			name:      "PubSubSubscription missing param",
			assetType: PubSubSubscription,
			params:    []string{""},
			expectErr: true,
		},
		{
			name:        "PubSubTopic valid",
			assetType:   PubSubTopic,
			params:      []string{"proj"},
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/topics",
			expectErr:   false,
		},
		{
			name:      "PubSubTopic missing param",
			assetType: PubSubTopic,
			params:    []string{""},
			expectErr: true,
		},
		{
			name:        "ComputeInstance valid",
			assetType:   ComputeInstance,
			params:      []string{"proj", "zone"},
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/zones/zone/instances",
			expectErr:   false,
		},
		{
			name:      "ComputeInstance missing zone",
			assetType: ComputeInstance,
			params:    []string{"proj", ""},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := SDPAssetTypeToAdapterMeta[tt.assetType]
			if !ok {
				t.Fatalf("assetType %v not found in SDPAssetTypeToAdapterMeta", tt.assetType)
			}
			if meta.ListEndpointFunc == nil {
				t.Skip("ListEndpointFunc not defined for this asset type")
			}
			gotURL, err := meta.ListEndpointFunc(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, gotURL)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, gotURL, tt.expectedURL)
				return
			}
			if gotURL != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", gotURL, tt.expectedURL)
			}
		})
	}
}

func TestSDPAssetTypeToAdapterMeta_SearchEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		assetType   shared.ItemType
		params      []string
		query       string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "BigQueryTable valid",
			assetType:   BigQueryTable,
			params:      []string{"proj"},
			query:       "dataset",
			expectedURL: "https://bigquery.googleapis.com/bigquery/v2/projects/proj/datasets/dataset/tables",
			expectErr:   false,
		},
		{
			name:      "BigQueryTable missing param",
			assetType: BigQueryTable,
			params:    []string{""},
			query:     "dataset",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := SDPAssetTypeToAdapterMeta[tt.assetType]
			if !ok {
				t.Fatalf("assetType %v not found in SDPAssetTypeToAdapterMeta", tt.assetType)
			}
			if meta.SearchEndpointFunc == nil {
				t.Skip("SearchEndpointFunc not defined for this asset type")
			}
			urlFunc, err := meta.SearchEndpointFunc(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, urlFunc)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, urlFunc, tt.expectedURL)
				return
			}
			gotURL := urlFunc(tt.query)
			if gotURL != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", gotURL, tt.expectedURL)
			}
		})
	}
}
