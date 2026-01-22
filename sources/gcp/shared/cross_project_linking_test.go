package shared

import (
	"reflect"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

// TestProjectBaseLinkedItemQueryByName_CrossProject verifies that project-level
// resources correctly extract the project ID from cross-project URIs
func TestProjectBaseLinkedItemQueryByName_CrossProject(t *testing.T) {
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: false,
	}

	tests := []struct {
		name        string
		projectID   string
		query       string
		want        *sdp.LinkedItemQuery
		description string
	}{
		{
			name:        "Same project - simple resource name",
			projectID:   "my-project",
			query:       "my-image",
			description: "Simple resource name without project prefix",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeImage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-image",
					Scope:  "my-project",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:        "Same project - full resource URI",
			projectID:   "my-project",
			query:       "projects/my-project/global/images/my-image",
			description: "Full resource URI with same project as context",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeImage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-image",
					Scope:  "my-project",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:        "Cross-project - full resource URI",
			projectID:   "box-dev-clamav",
			query:       "projects/box-dev-baseos/global/images/family/pcs-clamav-box",
			description: "Cross-project reference - should extract project from URI (ENG-2271 bug fix)",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeImage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "pcs-clamav-box",
					Scope:  "box-dev-baseos", // Should use extracted project, not context project
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:        "Cross-project - HTTPS URL",
			projectID:   "my-project",
			query:       "https://www.googleapis.com/compute/v1/projects/other-project/global/images/other-image",
			description: "Cross-project reference with HTTPS URL",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeImage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "other-image",
					Scope:  "other-project", // Should use extracted project, not context project
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:        "Empty query",
			projectID:   "my-project",
			query:       "",
			description: "Empty query should return nil",
			want:        nil,
		},
		{
			name:        "Empty project ID",
			projectID:   "",
			query:       "my-image",
			description: "Empty project ID should return nil",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkerFunc := ProjectBaseLinkedItemQueryByName(ComputeImage)
			got := linkerFunc(tt.projectID, "", tt.query, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProjectBaseLinkedItemQueryByName() = %v, want %v\nDescription: %s", got, tt.want, tt.description)
			}
		})
	}
}

// TestRegionBaseLinkedItemQueryByName_CrossProject verifies that regional
// resources correctly extract the project ID from cross-project URIs
func TestRegionBaseLinkedItemQueryByName_CrossProject(t *testing.T) {
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: false,
	}

	tests := []struct {
		name          string
		projectID     string
		fromItemScope string
		query         string
		want          *sdp.LinkedItemQuery
		description   string
	}{
		{
			name:          "Same project - full resource URI",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1",
			query:         "projects/my-project/regions/us-central1/addresses/my-address",
			description:   "Regional resource with same project",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeAddress.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-address",
					Scope:  "my-project.us-central1",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:          "Cross-project - full resource URI",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1",
			query:         "projects/other-project/regions/europe-west1/addresses/other-address",
			description:   "Cross-project regional resource - should extract project from URI",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeAddress.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "other-address",
					Scope:  "other-project.europe-west1", // Should use extracted project, not context project
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:          "Empty query",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1",
			query:         "",
			description:   "Empty query should return nil",
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkerFunc := RegionBaseLinkedItemQueryByName(ComputeAddress)
			got := linkerFunc(tt.projectID, tt.fromItemScope, tt.query, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RegionBaseLinkedItemQueryByName() = %v, want %v\nDescription: %s", got, tt.want, tt.description)
			}
		})
	}
}

// TestZoneBaseLinkedItemQueryByName_CrossProject verifies that zonal
// resources correctly extract the project ID from cross-project URIs
func TestZoneBaseLinkedItemQueryByName_CrossProject(t *testing.T) {
	blastPropagation := &sdp.BlastPropagation{
		In:  true,
		Out: false,
	}

	tests := []struct {
		name          string
		projectID     string
		fromItemScope string
		query         string
		want          *sdp.LinkedItemQuery
		description   string
	}{
		{
			name:          "Same project - full resource URI",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1-a",
			query:         "projects/my-project/zones/us-central1-a/disks/my-disk",
			description:   "Zonal resource with same project",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeDisk.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "my-disk",
					Scope:  "my-project.us-central1-a",
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:          "Cross-project - full resource URI",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1-a",
			query:         "projects/other-project/zones/europe-west1-b/disks/other-disk",
			description:   "Cross-project zonal resource - should extract project from URI",
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeDisk.String(),
					Method: sdp.QueryMethod_GET,
					Query:  "other-disk",
					Scope:  "other-project.europe-west1-b", // Should use extracted project, not context project
				},
				BlastPropagation: blastPropagation,
			},
		},
		{
			name:          "Empty query",
			projectID:     "my-project",
			fromItemScope: "my-project.us-central1-a",
			query:         "",
			description:   "Empty query should return nil",
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkerFunc := ZoneBaseLinkedItemQueryByName(ComputeDisk)
			got := linkerFunc(tt.projectID, tt.fromItemScope, tt.query, blastPropagation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZoneBaseLinkedItemQueryByName() = %v, want %v\nDescription: %s", got, tt.want, tt.description)
			}
		})
	}
}
