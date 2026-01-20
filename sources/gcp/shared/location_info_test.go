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
