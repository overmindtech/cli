package dynamic

import (
	"net/http"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
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
				ListEndpointFunc: func(loc gcpshared.LocationInfo) (string, error) { return "", nil },
			},
			want: Listable,
		},
		{
			name: "Searchable only",
			meta: gcpshared.AdapterMeta{
				SearchEndpointFunc: func(query string, loc gcpshared.LocationInfo) string {
					return ""
				},
			},
			want: Searchable,
		},
		{
			name: "SearchableListable",
			meta: gcpshared.AdapterMeta{
				ListEndpointFunc: func(loc gcpshared.LocationInfo) (string, error) { return "", nil },
				SearchEndpointFunc: func(query string, loc gcpshared.LocationInfo) string {
					return ""
				},
			},
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
		locations          []gcpshared.LocationInfo
		listable           bool
		searchable         bool
		searchableListable bool
		standard           bool
	}
	projectLocation := []gcpshared.LocationInfo{gcpshared.NewProjectLocation("my-project")}
	testCases := []testCase{
		{
			name:      "Listable adapter",
			sdpType:   gcpshared.ComputeFirewall,
			locations: projectLocation,
			listable:  true,
		},
		{
			name:       "Searchable adapter",
			sdpType:    gcpshared.SQLAdminBackupRun,
			locations:  projectLocation,
			searchable: true,
		},
		{
			name:               "SearchableListable adapter",
			sdpType:            gcpshared.MonitoringCustomDashboard,
			locations:          projectLocation,
			searchableListable: true,
		},
		{
			name:      "Standard adapter",
			sdpType:   gcpshared.CloudBillingBillingInfo,
			locations: projectLocation,
			standard:  true,
		},
	}

	linker := gcpshared.NewLinker()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meta := gcpshared.SDPAssetTypeToAdapterMeta[tc.sdpType]

			adapter, err := MakeAdapter(tc.sdpType, linker, http.DefaultClient, sdpcache.NewNoOpCache(), tc.locations)
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

func TestAdapters(t *testing.T) {
	type validator interface {
		Validate() error
	}

	// Let's ensure that we can create adapters without any issues.
	adapters, err := Adapters(
		[]gcpshared.LocationInfo{gcpshared.NewProjectLocation("my-project")},
		[]gcpshared.LocationInfo{gcpshared.NewRegionalLocation("my-project", "us-central1")},
		[]gcpshared.LocationInfo{gcpshared.NewZonalLocation("my-project", "us-central1-a")},
		gcpshared.NewLinker(),
		http.DefaultClient,
		nil,
		sdpcache.NewNoOpCache(),
	)
	if err != nil {
		t.Fatalf("Adapters() error = %v", err)
	}

	for _, adapter := range adapters {
		if adapter == nil {
			t.Error("Expected non-nil adapter, got nil")
			continue
		}

		meta := adapter.Metadata()
		if meta == nil {
			t.Error("Expected non-nil metadata, got nil")
			continue
		}

		validatable, ok := adapter.(validator)
		if !ok {
			t.Errorf("Expected adapter to implement Validate(), got %T", adapter)
			continue
		}

		if err := validatable.Validate(); err != nil {
			t.Errorf("Validate() error for adapter %s: %v", adapter.Name(), err)
		}

		if adapter.Metadata().GetTerraformMappings() != nil {
			for _, tm := range adapter.Metadata().GetTerraformMappings() {
				if tm.GetTerraformMethod() == sdp.QueryMethod_SEARCH {
					if _, ok := adapter.(discovery.SearchableAdapter); !ok {
						t.Errorf("Adapter %s has terraform mapping for SEARCH but does not implement SearchableAdapter", adapter.Name())
					}
				}
			}
		}
	}
}
