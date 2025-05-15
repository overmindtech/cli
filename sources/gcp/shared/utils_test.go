package shared_test

import (
	"testing"

	"github.com/overmindtech/cli/sources/gcp/shared"
)

func TestExtractRegion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid input with region",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default",
			expected: "us-central1",
		},
		{
			name:     "Valid input with different region",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/europe-west1/subnetworks/default",
			expected: "europe-west1",
		},
		{
			name:     "Valid input shortened",
			input:    "regions/region/subnetworks/subnetwork",
			expected: "region",
		},
		{
			name:     "Input without regions",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-a/instances/instance-1",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Malformed input",
			input:    "invalid-string-without-regions",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.ExtractRegion(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractRegion(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractZone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid input with zone",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-a/disks/integration-test-instance",
			expected: "us-central1-a",
		},
		{
			name:     "Valid input with different zone",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/europe-west1-b/disks/integration-test-instance",
			expected: "europe-west1-b",
		},
		{
			name:     "Valid input shortened",
			input:    "zones/zone/disks/disk",
			expected: "zone",
		},
		{
			name:     "Input without zones",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Malformed input",
			input:    "invalid-string-without-zones",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.ExtractZone(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractZone(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
