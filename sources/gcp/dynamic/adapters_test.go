package dynamic

import (
	"net/http"
	"testing"

	"github.com/overmindtech/cli/discovery"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func Test_adapterType(t *testing.T) {
	tests := []struct {
		name string
		meta gcpshared.AdapterMeta
		want typeOfAdapter
	}{
		{
			name: "Listable only",
			meta: gcpshared.AdapterMeta{
				ListEndpointFunc: func(queryParts ...string) (string, error) { return "", nil },
			},
			want: Listable,
		},
		{
			name: "Searchable only",
			meta: gcpshared.AdapterMeta{
				SearchEndpointFunc: func(queryParts ...string) (gcpshared.EndpointFunc, error) {
					return nil, nil
				},
			},
			want: Searchable,
		},
		{
			name: "SearchableListable",
			meta: gcpshared.AdapterMeta{
				ListEndpointFunc: func(queryParts ...string) (string, error) { return "", nil },
				SearchEndpointFunc: func(queryParts ...string) (gcpshared.EndpointFunc, error) {
					return nil, nil
				}},
			want: SearchableListable,
		},
		{
			name: "Standard (neither func set)",
			meta: gcpshared.AdapterMeta{},
			want: Standard,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adapterType(tt.meta); got != tt.want {
				t.Errorf("adapterType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addAdapter(t *testing.T) {
	type testCase struct {
		name               string
		sdpType            shared.ItemType
		opts               []string
		listable           bool
		searchable         bool
		searchableListable bool
		standard           bool
	}
	testCases := []testCase{
		{
			name:     "Listable adapter",
			sdpType:  gcpshared.ComputeInstance,
			opts:     []string{"my-project", "us-central1-a"},
			listable: true,
		},
		{
			name:       "Searchable adapter",
			sdpType:    gcpshared.SQLAdminBackupRun,
			opts:       []string{"my-project"},
			searchable: true,
		},
		{
			name:               "SearchableListable adapter",
			sdpType:            gcpshared.MonitoringCustomDashboard,
			opts:               []string{"my-project"},
			searchableListable: true,
		},
		{
			name:     "Standard adapter",
			sdpType:  gcpshared.CloudBillingBillingInfo,
			opts:     []string{"my-project"},
			standard: true,
		},
	}

	linker := gcpshared.NewLinker()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meta := gcpshared.SDPAssetTypeToAdapterMeta[tc.sdpType]

			adapter, err := MakeAdapter(tc.sdpType, meta, linker, http.DefaultClient, tc.opts...)
			if err != nil {
				t.Errorf("MakeAdapter() error = %v", err)
			}

			if tc.listable {
				if meta.ListEndpointFunc == nil {
					t.Errorf("Expected ListEndpointFunc to be set for listable adapter %s", tc.sdpType)
				}

				if meta.SearchEndpointFunc != nil {
					t.Errorf("Expected SearchEndpointFunc to be nil for listable adapter %s", tc.sdpType)
				}

				_, ok := adapter.(discovery.ListableAdapter)
				if !ok {
					t.Errorf("Expected adapter to be ListableAdapter, got %T", adapter)
				}

				return
			}

			if tc.searchable {
				if meta.SearchEndpointFunc == nil {
					t.Errorf("Expected SearchEndpointFunc to be set for searchable adapter %s", tc.sdpType)
				}

				if meta.ListEndpointFunc != nil {
					t.Errorf("Expected ListEndpointFunc to be nil for searchable adapter %s", tc.sdpType)
				}

				_, ok := adapter.(discovery.SearchableAdapter)
				if !ok {
					t.Errorf("Expected adapter to be SearchableAdapter, got %T", adapter)
				}

				return
			}

			if tc.searchableListable {
				if meta.ListEndpointFunc == nil {
					t.Errorf("Expected ListEndpointFunc to be set for searchable listable adapter %s", tc.sdpType)
				}

				if meta.SearchEndpointFunc == nil {
					t.Errorf("Expected SearchEndpointFunc to be set for searchable listable adapter %s", tc.sdpType)
				}

				_, ok := adapter.(SearchableListableAdapter)
				if !ok {
					t.Errorf("Expected adapter to be SearchableListableAdapter, got %T", adapter)
				}

				return
			}

			if tc.standard {
				if meta.ListEndpointFunc != nil {
					t.Errorf("Expected ListEndpointFunc to be nil for standard adapter %s", tc.sdpType)
				}

				if meta.SearchEndpointFunc != nil {
					t.Errorf("Expected SearchEndpointFunc to be nil for standard adapter %s", tc.sdpType)
				}

				return
			}
		})
	}
}
