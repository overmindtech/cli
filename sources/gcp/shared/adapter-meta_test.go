package shared

import (
	"testing"

	"github.com/overmindtech/cli/sources/shared"
)

func TestSDPAssetTypeToAdapterMeta_GetEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		assetType   shared.ItemType
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "ComputeNetwork valid",
			assetType:   ComputeNetwork,
			location:    NewProjectLocation("proj"),
			query:       "net",
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/global/networks/net",
		},
		{
			name:        "ComputeSubnetwork valid",
			assetType:   ComputeSubnetwork,
			location:    NewRegionalLocation("proj", "region"),
			query:       "subnet",
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/regions/region/subnetworks/subnet",
		},
		{
			name:        "PubSubSubscription valid",
			assetType:   PubSubSubscription,
			location:    NewProjectLocation("proj"),
			query:       "mysub",
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/subscriptions/mysub",
		},
		{
			name:        "PubSubTopic valid",
			assetType:   PubSubTopic,
			location:    NewProjectLocation("proj"),
			query:       "mytopic",
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/topics/mytopic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, ok := SDPAssetTypeToAdapterMeta[tt.assetType]
			if !ok {
				t.Fatalf("assetType %v not found in SDPAssetTypeToAdapterMeta", tt.assetType)
			}
			if meta.GetEndpointFunc == nil {
				t.Fatalf("GetEndpointFunc is nil for asset type %v", tt.assetType)
			}
			gotURL := meta.GetEndpointFunc(tt.query, tt.location)
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
		location    LocationInfo
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "ComputeNetwork valid",
			assetType:   ComputeNetwork,
			location:    NewProjectLocation("proj"),
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/global/networks",
		},
		{
			name:      "ComputeNetwork missing param",
			assetType: ComputeNetwork,
			location:  LocationInfo{},
			expectErr: true,
		},
		{
			name:        "ComputeSubnetwork valid",
			assetType:   ComputeSubnetwork,
			location:    NewRegionalLocation("proj", "region"),
			expectedURL: "https://compute.googleapis.com/compute/v1/projects/proj/regions/region/subnetworks",
		},
		{
			name:      "ComputeSubnetwork missing region",
			assetType: ComputeSubnetwork,
			location:  NewProjectLocation("proj"),
			expectErr: true,
		},
		{
			name:        "PubSubSubscription valid",
			assetType:   PubSubSubscription,
			location:    NewProjectLocation("proj"),
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/subscriptions",
		},
		{
			name:        "PubSubTopic valid",
			assetType:   PubSubTopic,
			location:    NewProjectLocation("proj"),
			expectedURL: "https://pubsub.googleapis.com/v1/projects/proj/topics",
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
			gotURL, err := meta.ListEndpointFunc(tt.location)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none\n  got: %v", gotURL)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
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
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "ArtifactRegistryDockerImage valid",
			assetType:   ArtifactRegistryDockerImage,
			location:    NewProjectLocation("my-project"),
			query:       "my-location|my-repo",
			expectedURL: "https://artifactregistry.googleapis.com/v1/projects/my-project/locations/my-location/repositories/my-repo/dockerImages",
		},
		{
			name:        "ArtifactRegistryDockerImage invalid query returns empty",
			assetType:   ArtifactRegistryDockerImage,
			location:    NewProjectLocation("my-project"),
			query:       "my-location", // Missing repo part
			expectedURL: "",
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
			gotURL := meta.SearchEndpointFunc(tt.query, tt.location)
			if gotURL != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", gotURL, tt.expectedURL)
			}
		})
	}
}

func TestProjectLevelGetEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "valid project and query",
			format:      "https://example.com/projects/%s/resources/%s",
			location:    NewProjectLocation("my-project"),
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/resources/my-resource",
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/resources/%s",
			location:    NewProjectLocation("my-project"),
			query:       "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointFunc := ProjectLevelEndpointFuncWithSingleQuery(tt.format)
			got := endpointFunc(tt.query, tt.location)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestProjectLevelGetEndpointFuncWithTwoQueries(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "valid project and composite query",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			location:    NewProjectLocation("my-project"),
			query:       "foo|bar",
			expectedURL: "https://example.com/projects/my-project/parent-resources/foo/child-resources/bar",
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			location:    NewProjectLocation("my-project"),
			query:       "",
			expectedURL: "",
		},
		{
			name:        "query with only one part returns empty string",
			format:      "https://example.com/projects/%s/parent-resources/%s/child-resources/%s",
			location:    NewProjectLocation("my-project"),
			query:       "foo",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointFunc := ProjectLevelEndpointFuncWithTwoQueries(tt.format)
			got := endpointFunc(tt.query, tt.location)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestZoneLevelGetEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "valid project, zone and query",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s",
			location:    NewZonalLocation("my-project", "my-zone"),
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/zones/my-zone/resources/my-resource",
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/zones/%s/resources/%s",
			location:    NewZonalLocation("my-project", "my-zone"),
			query:       "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointFunc := ZoneLevelEndpointFunc(tt.format)
			got := endpointFunc(tt.query, tt.location)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestRegionalLevelGetEndpointFunc(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		location    LocationInfo
		query       string
		expectedURL string
	}{
		{
			name:        "valid project, region and query",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s",
			location:    NewRegionalLocation("my-project", "my-region"),
			query:       "my-resource",
			expectedURL: "https://example.com/projects/my-project/regions/my-region/resources/my-resource",
		},
		{
			name:        "empty query returns empty string",
			format:      "https://example.com/projects/%s/regions/%s/resources/%s",
			location:    NewRegionalLocation("my-project", "my-region"),
			query:       "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointFunc := RegionalLevelEndpointFunc(tt.format)
			got := endpointFunc(tt.query, tt.location)
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}

func TestEndpointFuncWithQueries_PanicsOnWrongFormat(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) EndpointFunc
		format string
	}{
		{
			name:   "ProjectLevelGetEndpointFuncWithThreeQueries panics on wrong format",
			fn:     ProjectLevelEndpointFuncWithThreeQueries,
			format: "https://example.com/projects/%s/resources/%s/child/%s", // 3 %s, should be 4
		},
		{
			name:   "ProjectLevelGetEndpointFunc panics on wrong format",
			fn:     ProjectLevelEndpointFuncWithSingleQuery,
			format: "https://example.com/projects/%s/resources", // 1 %s, should be 2
		},
		{
			name:   "ZoneLevelGetEndpointFunc panics on wrong format",
			fn:     ZoneLevelEndpointFunc,
			format: "https://example.com/projects/%s/zones/%s/resources", // 2 %s, should be 3
		},
		{
			name:   "RegionalLevelGetEndpointFunc panics on wrong format",
			fn:     RegionalLevelEndpointFunc,
			format: "https://example.com/projects/%s/regions/%s/resources", // 2 %s, should be 3
		},
	}

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
		location    LocationInfo
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project id",
			format:      "https://example.com/projects/%s/resources",
			location:    NewProjectLocation("my-project"),
			expectedURL: "https://example.com/projects/my-project/resources",
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/resources",
			location:  LocationInfo{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := ProjectLevelListFunc(tt.format)
			got, err := fn(tt.location)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none\n  got: %v", got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
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
		location    LocationInfo
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project and region",
			format:      "https://example.com/projects/%s/regions/%s/resources",
			location:    NewRegionalLocation("my-project", "my-region"),
			expectedURL: "https://example.com/projects/my-project/regions/my-region/resources",
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			location:  LocationInfo{Region: "my-region"},
			expectErr: true,
		},
		{
			name:      "empty region",
			format:    "https://example.com/projects/%s/regions/%s/resources",
			location:  LocationInfo{ProjectID: "my-project"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := RegionLevelListFunc(tt.format)
			got, err := fn(tt.location)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none\n  got: %v", got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
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
		location    LocationInfo
		expectedURL string
		expectErr   bool
	}{
		{
			name:        "valid project and zone",
			format:      "https://example.com/projects/%s/zones/%s/resources",
			location:    NewZonalLocation("my-project", "my-zone"),
			expectedURL: "https://example.com/projects/my-project/zones/my-zone/resources",
		},
		{
			name:      "empty project id",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			location:  LocationInfo{Zone: "my-zone"},
			expectErr: true,
		},
		{
			name:      "empty zone",
			format:    "https://example.com/projects/%s/zones/%s/resources",
			location:  LocationInfo{ProjectID: "my-project"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := ZoneLevelListFunc(tt.format)
			got, err := fn(tt.location)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none\n  got: %v", got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.expectedURL {
				t.Errorf("unexpected URL:\n  got:  %v\n  want: %v", got, tt.expectedURL)
			}
		})
	}
}
