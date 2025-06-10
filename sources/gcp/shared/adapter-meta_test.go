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

func TestProjectLevelEndpointFuncWithSingleQuery(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project and query",
			format:      "https://example.com/projects/%s/resources/%s",
			params:      []string{"my-project"},
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/resources/my-resource",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/resources/%s",
			params:        []string{""},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/resources/%s",
			params:      []string{"my-project"},
			query:       "",
			expectedURL: "",
		},
		{
			name:          "too many init params",
			format:        "https://example.com/projects/%s/resources/%s",
			params:        []string{"my-project", "extra"},
			query:         "my-resource",
			expectInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := projectLevelEndpointFuncWithSingleQuery(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)", tt.params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)", err, tt.params)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestProjectLevelEndpointFuncWithTwoQueries(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project and composite query",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:      []string{"my-project"},
			query:       "foo|bar",
			expectedURL: "https://example.com/projects/my-project/parent-resources/foo/child-resources/bar",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:        []string{""},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:      []string{"my-project"},
			query:       "",
			expectedURL: "",
		},
		{
			name:        "query with only one part returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:      []string{"my-project"},
			query:       "foo",
			expectedURL: "",
		},
		{
			name:        "query with empty part returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:      []string{"my-project"},
			query:       "foo|",
			expectedURL: "",
		},
		{
			name:        "query with both parts empty returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:      []string{"my-project"},
			query:       "|",
			expectedURL: "",
		},
		{
			name:          "too many init params",
			format:        "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			params:        []string{"my-project", "extra"},
			query:         "foo|bar",
			expectInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := projectLevelEndpointFuncWithTwoQueries(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)", tt.params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)", err, tt.params)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestZoneLevelEndpointFuncWithSingleQuery(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project, zone and query",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/zones/my-zone/resources/my-resource",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s",
			params:        []string{"", "my-zone"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "empty zone",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s",
			params:        []string{"my-project", ""},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "too few params",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s",
			params:        []string{"my-project"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "too many params",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s",
			params:        []string{"my-project", "my-zone", "extra"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := zoneLevelEndpointFuncWithSingleQuery(tt.format)(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)", tt.params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)", err, tt.params)
			}
			got := fn(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestRegionalLevelEndpointFuncWithSingleQuery(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project, region and query",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s",
			params:      []string{"my-project", "my-region"},
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/regions/my-region/resources/my-resource",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s",
			params:        []string{"", "my-region"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "empty region",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s",
			params:        []string{"my-project", ""},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "too few params",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s",
			params:        []string{"my-project"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:          "too many params",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s",
			params:        []string{"my-project", "my-region", "extra"},
			query:         "my-resource",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s",
			params:      []string{"my-project", "my-region"},
			query:       "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := regionalLevelEndpointFuncWithSingleQuery(tt.format)(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)", tt.params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)", err, tt.params)
			}
			got := fn(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestZoneLevelEndpointFuncWithTwoQueries(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project, zone and composite query",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "foo|bar",
			expectedURL: "https://example.com/projects/my-project/zones/my-zone/resources/foo/child/bar",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:        []string{"", "my-zone"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "empty zone",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:        []string{"my-project", ""},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "too few params",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:        []string{"my-project"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "too many params",
			format:        "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:        []string{"my-project", "my-zone", "extra"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "",
			expectedURL: "",
		},
		{
			name:        "query with only one part returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "foo",
			expectedURL: "",
		},
		{
			name:        "query with empty part returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "foo|",
			expectedURL: "",
		},
		{
			name:        "query with both parts empty returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-zone"},
			query:       "|",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := zoneLevelEndpointFuncWithTwoQueries(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, endpointFunc)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, endpointFunc, tt.expectedURL)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestRegionalLevelEndpointFuncWithTwoQueries(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project, region and composite query",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-region"},
			query:       "foo|bar",
			expectedURL: "https://example.com/projects/my-project/regions/my-region/resources/foo/child/bar",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:        []string{"", "my-region"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "empty region",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:        []string{"my-project", ""},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "too few params",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:        []string{"my-project"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:          "too many params",
			format:        "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:        []string{"my-project", "my-region", "extra"},
			query:         "foo|bar",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-region"},
			query:       "",
			expectedURL: "",
		},
		{
			name:        "query with only one part returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-region"},
			query:       "foo",
			expectedURL: "",
		},
		{
			name:        "query with empty part returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-region"},
			query:       "foo|",
			expectedURL: "",
		},
		{
			name:        "query with both parts empty returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s/child/%s",
			params:      []string{"my-project", "my-region"},
			query:       "|",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := regionalLevelEndpointFuncWithTwoQueries(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, endpointFunc)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, endpointFunc, tt.expectedURL)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestProjectLevelEndpointFuncWithThreeQueries(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project and triple composite query",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "foo|bar|baz",
			expectedURL: "https://example.com/projects/my-project/resources/foo/child/bar/grandchild/baz",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:        []string{""},
			query:         "foo|bar|baz",
			expectInitErr: true,
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "",
			expectedURL: "",
		},
		{
			name:        "query with only one part returns empty string",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "foo",
			expectedURL: "",
		},
		{
			name:        "query with two parts returns empty string",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "foo|bar",
			expectedURL: "",
		},
		{
			name:        "query with empty part returns empty string",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "foo|bar|",
			expectedURL: "",
		},
		{
			name:        "query with all parts empty returns empty string",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:      []string{"my-project"},
			query:       "||",
			expectedURL: "",
		},
		{
			name:          "too many init params",
			format:        "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s",
			params:        []string{"my-project", "extra"},
			query:         "foo|bar|baz",
			expectInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := projectLevelEndpointFuncWithThreeQueries(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, endpointFunc)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, endpointFunc, tt.expectedURL)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestEndpointFuncWithQueries_PanicsOnWrongFormat(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) func(...string) (EndpointFunc, error)
		format string
		count  int
	}{
		{
			name:   "projectLevelEndpointFuncWithThreeQueries panics on wrong format",
			fn:     projectLevelEndpointFuncWithThreeQueries,
			format: "https://example.com/projects/%s/resources/%s/child/%s", // 3 %s, should be 4
			count:  4,
		},
		{
			name:   "projectLevelEndpointFuncWithSingleQuery panics on wrong format",
			fn:     projectLevelEndpointFuncWithSingleQuery,
			format: "https://example.com/projects/%s/resources", // 1 %s, should be 2
			count:  2,
		},
		{
			name:   "projectLevelEndpointFuncWithTwoQueries panics on wrong format",
			fn:     projectLevelEndpointFuncWithTwoQueries,
			format: "https://example.com/projects/%s/resources/%s", // 2 %s, should be 3
			count:  3,
		},
		{
			name:   "zoneLevelEndpointFuncWithSingleQuery panics on wrong format",
			fn:     zoneLevelEndpointFuncWithSingleQuery,
			format: "https://example.com/projects/%s/zones/%s/resources", // 2 %s, should be 3
			count:  3,
		},
		{
			name:   "regionalLevelEndpointFuncWithSingleQuery panics on wrong format",
			fn:     regionalLevelEndpointFuncWithSingleQuery,
			format: "https://example.com/projects/%s/regions/%s/resources", // 2 %s, should be 3
			count:  3,
		},
		{
			name:   "zoneLevelEndpointFuncWithTwoQueries panics on wrong format",
			fn:     zoneLevelEndpointFuncWithTwoQueries,
			format: "https://example.com/projects/%s/zones/%s/resources/%s", // 3 %s, should be 4
			count:  4,
		},
		{
			name:   "regionalLevelEndpointFuncWithTwoQueries panics on wrong format",
			fn:     regionalLevelEndpointFuncWithTwoQueries,
			format: "https://example.com/projects/%s/regions/%s/resources/%s", // 3 %s, should be 4
			count:  4,
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for wrong format, but no panic occurred (format: %v)", tt.format)
				}
			}()
			_ = tt.fn(tt.format)
		})
	}
}

func Test_projectLevelListFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		params      []string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project id",
			format:      "https://example.com/projects/%s/resources",
			params:      []string{"my-project"},
			expectedURL: "https://example.com/projects/my-project/resources",
			expectErr:   false,
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/resources",
			params:    []string{""},
			expectErr: true,
		},
		{
			name:      "too many params",
			format:    "https://example.com/projects/%s/resources",
			params:    []string{"my-project", "extra"},
			expectErr: true,
		},
		{
			name:      "too few params",
			format:    "https://example.com/projects/%s/resources",
			params:    []string{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := projectLevelListFunc(tt.format)
			got, err := fn(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, got, tt.expectedURL)
				return
			}
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func Test_regionLevelListFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		params      []string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project and region",
			format:      "https://example.com/projects/%s/regions/%s/resources",
			params:      []string{"my-project", "my-region"},
			expectedURL: "https://example.com/projects/my-project/regions/my-region/resources",
			expectErr:   false,
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			params:    []string{"", "my-region"},
			expectErr: true,
		},
		{
			name:      "empty region",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			params:    []string{"my-project", ""},
			expectErr: true,
		},
		{
			name:      "too few params",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			params:    []string{"my-project"},
			expectErr: true,
		},
		{
			name:      "too many params",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			params:    []string{"my-project", "my-region", "extra"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := regionLevelListFunc(tt.format)
			got, err := fn(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, got, tt.expectedURL)
				return
			}
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func Test_zoneLevelListFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		params      []string
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project and zone",
			format:      "https://example.com/projects/%s/zones/%s/resources",
			params:      []string{"my-project", "my-zone"},
			expectedURL: "https://example.com/projects/my-project/zones/my-zone/resources",
			expectErr:   false,
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			params:    []string{"", "my-zone"},
			expectErr: true,
		},
		{
			name:      "empty zone",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			params:    []string{"my-project", ""},
			expectErr: true,
		},
		{
			name:      "too few params",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			params:    []string{"my-project"},
			expectErr: true,
		},
		{
			name:      "too many params",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			params:    []string{"my-project", "my-zone", "extra"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := zoneLevelListFunc(tt.format)
			got, err := fn(tt.params...)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, got, tt.expectedURL)
				return
			}
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func Test_projectLevelEndpointFuncWithFourQueries(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		params        []string
		query         string
		expectedURL   string
		expectInitErr bool
	}{
		{
			name:        "valid project and quadruple composite query",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s/greatgrandchild/%s",
			params:      []string{"my-project"},
			query:       "foo|bar|baz|qux",
			expectedURL: "https://example.com/projects/my-project/resources/foo/child/bar/grandchild/baz/greatgrandchild/qux",
		},
		{
			name:          "empty project id",
			format:        "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s/greatgrandchild/%s",
			params:        []string{""},
			query:         "foo|bar|baz|qux",
			expectInitErr: true,
		},
		{
			name:        "missing query",
			format:      "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s/greatgrandchild/%s",
			params:      []string{"my-project"},
			query:       "",
			expectedURL: "",
		},
		{
			name:          "too many params",
			format:        "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s/greatgrandchild/%s",
			params:        []string{"my-project", "extra"},
			query:         "foo|bar|baz|qux",
			expectInitErr: true,
		},
		{
			name:          "too few params",
			format:        "https://example.com/projects/%s/resources/%s/child/%s/grandchild/%s/greatgrandchild/%s",
			params:        []string{},
			query:         "foo|bar|baz|qux",
			expectInitErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := projectLevelEndpointFuncWithFourQueries(tt.format)
			endpointFunc, err := fn(tt.params...)
			if tt.expectInitErr {
				if err == nil {
					t.Errorf("expected error but got none (params: %v)\n  got: %v\n  want error", tt.params, endpointFunc)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v (params: %v)\n  got: %v\n  want: %v", err, tt.params, endpointFunc, tt.expectedURL)
			}
			got := endpointFunc(tt.query)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}
