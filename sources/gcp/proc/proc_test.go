package proc

import (
	"testing"
)

func TestZoneToRegion(t *testing.T) {
	tests := []struct {
		name     string
		zone     string
		expected string
	}{
		{
			name:     "Valid zone with region us-central1-a",
			zone:     "us-central1-a",
			expected: "us-central1",
		},
		{
			name:     "Valid zone with region europe-west1-b",
			zone:     "europe-west1-b",
			expected: "europe-west1",
		},
		{
			name:     "Empty zone",
			zone:     "",
			expected: "",
		},
		{
			name:     "Zone with no dash",
			zone:     "uscentral1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := zoneToRegion(tt.zone)
			if result != tt.expected {
				t.Errorf("zoneToRegion(%q) = %q; expected %q", tt.zone, result, tt.expected)
			}
		})
	}
}
