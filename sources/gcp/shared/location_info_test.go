package shared_test

import (
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestLocationFromScope(t *testing.T) {
	tests := []struct {
		name              string
		scope             string
		wantProjectID     string
		wantRegion        string
		wantZone          string
		wantLocationLevel gcpshared.LocationLevel
		wantErr           bool
	}{
		{
			name:              "project scope",
			scope:             "my-project",
			wantProjectID:     "my-project",
			wantLocationLevel: gcpshared.ProjectLevel,
		},
		{
			name:              "regional scope",
			scope:             "my-project.us-central1",
			wantProjectID:     "my-project",
			wantRegion:        "us-central1",
			wantLocationLevel: gcpshared.RegionalLevel,
		},
		{
			name:              "zonal scope",
			scope:             "my-project.us-central1-a",
			wantProjectID:     "my-project",
			wantRegion:        "us-central1",
			wantZone:          "us-central1-a",
			wantLocationLevel: gcpshared.ZonalLevel,
		},
		{
			name:    "empty scope",
			scope:   "",
			wantErr: true,
		},
		{
			name:    "invalid scope has too many parts",
			scope:   "a.b.c",
			wantErr: true,
		},
		{
			name:    "invalid location dash count",
			scope:   "my-project.global",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate that the parsed result is consistent with LocationInfo
			// Also validates using LocationFromScope for consistency
			locationInfo, parseErr := gcpshared.LocationFromScope(tt.scope)
			if tt.wantErr {
				if parseErr == nil {
					t.Fatalf("LocationFromScope(%q) expected error but got none", tt.scope)
				}
				return
			} else {
				if parseErr != nil {
					t.Fatalf("LocationFromScope(%q) unexpected error: %v", tt.scope, parseErr)
				}
			}

			// Validate the LocationInfo
			if validateErr := locationInfo.Validate(); validateErr != nil {
				t.Errorf("LocationInfo.Validate() failed for scope %q: %v", tt.scope, validateErr)
			}

			// Verify consistency between LocationFromScope and LocationInfo
			if locationInfo.ProjectID != tt.wantProjectID {
				t.Errorf("ProjectID mismatch: LocationPartsFromScope=%q, LocationFromScope=%q", tt.wantProjectID, locationInfo.ProjectID)
			}
			if locationInfo.Region != tt.wantRegion {
				t.Errorf("Region mismatch: LocationPartsFromScope=%q, LocationFromScope=%q", tt.wantRegion, locationInfo.Region)
			}
			if locationInfo.Zone != tt.wantZone {
				t.Errorf("Zone mismatch: LocationPartsFromScope=%q, LocationFromScope=%q", tt.wantZone, locationInfo.Zone)
			}
			if locationInfo.LocationLevel() != tt.wantLocationLevel {
				t.Errorf("ScopeType mismatch: LocationPartsFromScope=%q, LocationFromScope=%q", tt.wantLocationLevel, locationInfo.LocationLevel())
			}

			// Verify scope type detection is mutually exclusive
			switch tt.wantLocationLevel {
			case gcpshared.ProjectLevel:
				if locationInfo.Regional() || locationInfo.Zonal() {
					t.Errorf("Project scope should not be Regional or Zonal")
				}
				if !locationInfo.ProjectLevel() {
					t.Errorf("Project scope should be ProjectLevel")
				}
			case gcpshared.RegionalLevel:
				if !locationInfo.Regional() {
					t.Errorf("Regional scope should have Regional()=true")
				}
				if locationInfo.Zonal() || locationInfo.ProjectLevel() {
					t.Errorf("Regional scope should not be Zonal or ProjectLevel")
				}
			case gcpshared.ZonalLevel:
				if !locationInfo.Zonal() {
					t.Errorf("Zonal scope should have Zonal()=true")
				}
				if locationInfo.Regional() || locationInfo.ProjectLevel() {
					t.Errorf("Zonal scope should not be Regional or ProjectLevel")
				}
			}
		})
	}
}

func TestGetProjectIDsFromLocations(t *testing.T) {
	tests := []struct {
		name     string
		slices   [][]gcpshared.LocationInfo
		expected []string
	}{
		{
			name:     "empty slices",
			slices:   [][]gcpshared.LocationInfo{},
			expected: nil,
		},
		{
			name:     "single empty slice",
			slices:   [][]gcpshared.LocationInfo{{}},
			expected: nil,
		},
		{
			name: "single slice with one project",
			slices: [][]gcpshared.LocationInfo{
				{gcpshared.NewZonalLocation("project-a", "us-central1-a")},
			},
			expected: []string{"project-a"},
		},
		{
			name: "single slice with multiple locations same project",
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
					gcpshared.NewZonalLocation("project-a", "us-central1-b"),
					gcpshared.NewZonalLocation("project-a", "us-east1-a"),
				},
			},
			expected: []string{"project-a"},
		},
		{
			name: "single slice with multiple projects",
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
					gcpshared.NewZonalLocation("project-b", "us-central1-a"),
					gcpshared.NewZonalLocation("project-c", "us-east1-a"),
				},
			},
			expected: []string{"project-a", "project-b", "project-c"},
		},
		{
			name: "multiple slices with overlapping projects",
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewProjectLocation("project-a"),
					gcpshared.NewProjectLocation("project-b"),
				},
				{
					gcpshared.NewRegionalLocation("project-b", "us-central1"),
					gcpshared.NewRegionalLocation("project-c", "us-east1"),
				},
			},
			expected: []string{"project-a", "project-b", "project-c"},
		},
		{
			name: "multiple slices with no overlap",
			slices: [][]gcpshared.LocationInfo{
				{gcpshared.NewZonalLocation("project-a", "us-central1-a")},
				{gcpshared.NewRegionalLocation("project-b", "us-east1")},
			},
			expected: []string{"project-a", "project-b"},
		},
		{
			name: "preserves order of first occurrence",
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewProjectLocation("project-c"),
					gcpshared.NewProjectLocation("project-a"),
				},
				{
					gcpshared.NewProjectLocation("project-b"),
					gcpshared.NewProjectLocation("project-a"),
				},
			},
			expected: []string{"project-c", "project-a", "project-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.GetProjectIDsFromLocations(tt.slices...)
			if len(result) != len(tt.expected) {
				t.Errorf("GetProjectIDsFromLocations() returned %d items, expected %d. Got: %v, expected: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, projectID := range result {
				if projectID != tt.expected[i] {
					t.Errorf("GetProjectIDsFromLocations()[%d] = %q, expected %q", i, projectID, tt.expected[i])
				}
			}
		})
	}
}

func TestHasLocationInSlices(t *testing.T) {
	tests := []struct {
		name     string
		loc      gcpshared.LocationInfo
		slices   [][]gcpshared.LocationInfo
		expected bool
	}{
		{
			name:     "empty slices",
			loc:      gcpshared.NewZonalLocation("project-a", "us-central1-a"),
			slices:   [][]gcpshared.LocationInfo{},
			expected: false,
		},
		{
			name:     "single empty slice",
			loc:      gcpshared.NewZonalLocation("project-a", "us-central1-a"),
			slices:   [][]gcpshared.LocationInfo{{}},
			expected: false,
		},
		{
			name: "location in first slice",
			loc:  gcpshared.NewZonalLocation("project-a", "us-central1-a"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
					gcpshared.NewZonalLocation("project-a", "us-central1-b"),
				},
				{
					gcpshared.NewRegionalLocation("project-b", "us-east1"),
				},
			},
			expected: true,
		},
		{
			name: "location in second slice",
			loc:  gcpshared.NewRegionalLocation("project-b", "us-east1"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
				},
				{
					gcpshared.NewRegionalLocation("project-b", "us-east1"),
				},
			},
			expected: true,
		},
		{
			name: "location in neither slice",
			loc:  gcpshared.NewZonalLocation("project-c", "us-west1-a"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
				},
				{
					gcpshared.NewRegionalLocation("project-b", "us-east1"),
				},
			},
			expected: false,
		},
		{
			name: "matching project but different region",
			loc:  gcpshared.NewRegionalLocation("project-a", "us-east1"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewRegionalLocation("project-a", "us-central1"),
				},
			},
			expected: false,
		},
		{
			name: "matching project and region but different zone",
			loc:  gcpshared.NewZonalLocation("project-a", "us-central1-b"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewZonalLocation("project-a", "us-central1-a"),
				},
			},
			expected: false,
		},
		{
			name: "exact match for project-level location",
			loc:  gcpshared.NewProjectLocation("project-a"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewProjectLocation("project-a"),
					gcpshared.NewProjectLocation("project-b"),
				},
			},
			expected: true,
		},
		{
			name: "project-level location not found when only regional exists",
			loc:  gcpshared.NewProjectLocation("project-a"),
			slices: [][]gcpshared.LocationInfo{
				{
					gcpshared.NewRegionalLocation("project-a", "us-central1"),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.HasLocationInSlices(tt.loc, tt.slices...)
			if result != tt.expected {
				t.Errorf("HasLocationInSlices(%v, ...) = %v, expected %v", tt.loc, result, tt.expected)
			}
		})
	}
}
