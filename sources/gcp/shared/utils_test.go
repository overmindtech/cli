package shared_test

import (
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestLastPathComponent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "projects/test-project/zones/us-central1-a/disks/my-disk",
			expected: "my-disk",
		},
		{
			input:    "projects/test-project/zones/us-central1-a",
			expected: "us-central1-a",
		},
		{
			input:    "my-disk",
			expected: "my-disk",
		},
		{
			input:    "",
			expected: "",
		},
		{
			input:    "/",
			expected: "",
		},
		{
			input:    "////",
			expected: "",
		},
		{
			input:    "foo/bar/baz",
			expected: "baz",
		},
	}

	for _, tc := range tests {
		actual := gcpshared.LastPathComponent(tc.input)
		if actual != tc.expected {
			t.Errorf("LastPathComponent(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestIsRegion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid region",
			input:    "us-central1",
			expected: true,
		},
		{
			name:     "another valid region",
			input:    "asia-east1",
			expected: true,
		},
		{
			name:     "zone, not region",
			input:    "us-central1-a",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "no hyphen",
			input:    "uscentral1",
			expected: false,
		},
		{
			name:     "too many hyphens",
			input:    "us-central-1",
			expected: false,
		},
		{
			name:     "just a hyphen",
			input:    "-",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.IsRegion(tt.input)
			if result != tt.expected {
				t.Errorf("IsRegion(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsZone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid zone",
			input:    "us-central1-a",
			expected: true,
		},
		{
			name:     "another valid zone",
			input:    "asia-east1-b",
			expected: true,
		},
		{
			name:     "region, not zone",
			input:    "us-central1",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "no hyphen",
			input:    "uscentral1a",
			expected: false,
		},
		{
			name:     "too many hyphens",
			input:    "us-central-1-a",
			expected: false,
		},
		{
			name:     "just hyphens",
			input:    "--",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.IsZone(tt.input)
			if result != tt.expected {
				t.Errorf("IsZone(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		input    string
		expected string
	}{
		// ExtractLocation cases
		{
			name:     "ExtractLocation: Valid input with location",
			key:      "locations",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key/cryptoKeyVersions/3",
			expected: "us-central1",
		},
		{
			name:     "ExtractLocation: Different region",
			key:      "locations",
			input:    "projects/proj/locations/europe-west1/keyRings/ring/cryptoKeys/key/cryptoKeyVersions/5",
			expected: "europe-west1",
		},
		{
			name:     "ExtractLocation: No location in path",
			key:      "locations",
			input:    "projects/proj/keyRings/ring/cryptoKeys/key",
			expected: "",
		},
		{
			name:     "ExtractLocation: Empty input",
			key:      "locations",
			input:    "",
			expected: "",
		},
		{
			name:     "ExtractLocation: Malformed input",
			key:      "locations",
			input:    "this-is-not-a-kms-path",
			expected: "",
		},

		// ExtractKeyRing cases
		{
			name:     "ExtractKeyRing: Valid input with key ring",
			key:      "keyRings",
			input:    "projects/proj/locations/us/keyRings/ring/cryptoKeys/key/cryptoKeyVersions/1",
			expected: "ring",
		},
		{
			name:     "ExtractKeyRing: Different key ring",
			key:      "keyRings",
			input:    "projects/proj/locations/europe/keyRings/test-ring/cryptoKeys/key/cryptoKeyVersions/1",
			expected: "test-ring",
		},
		{
			name:     "ExtractKeyRing: Missing keyRings segment",
			key:      "keyRings",
			input:    "projects/proj/locations/loc/cryptoKeys/key",
			expected: "",
		},
		{
			name:     "ExtractKeyRing: Empty input",
			key:      "keyRings",
			input:    "",
			expected: "",
		},
		{
			name:     "ExtractKeyRing: Malformed path",
			key:      "keyRings",
			input:    "keyRings",
			expected: "",
		},

		// ExtractCryptoKey cases
		{
			name:     "ExtractCryptoKey: Valid input",
			key:      "cryptoKeys",
			input:    "projects/proj/locations/loc/keyRings/ring/cryptoKeys/key/cryptoKeyVersions/1",
			expected: "key",
		},
		{
			name:     "ExtractCryptoKey: Another valid input",
			key:      "cryptoKeys",
			input:    "projects/a/locations/b/keyRings/r/cryptoKeys/my-key/cryptoKeyVersions/2",
			expected: "my-key",
		},
		{
			name:     "ExtractCryptoKey: Missing cryptoKeys segment",
			key:      "cryptoKeys",
			input:    "projects/p/locations/l/keyRings/r/cryptoKeyVersions/1",
			expected: "",
		},
		{
			name:     "ExtractCryptoKey: Empty input",
			key:      "cryptoKeys",
			input:    "",
			expected: "",
		},
		{
			name:     "ExtractCryptoKey: Malformed string",
			key:      "cryptoKeys",
			input:    "cryptoKeyVersions",
			expected: "",
		},

		// ExtractCryptoKeyVersion cases (as ExtractResourcePart)
		{
			name:     "ExtractCryptoKeyVersion: Valid input",
			key:      "cryptoKeyVersions",
			input:    "projects/proj/locations/loc/keyRings/ring/cryptoKeys/key/cryptoKeyVersions/3",
			expected: "3",
		},
		{
			name:     "ExtractCryptoKeyVersion: Different version",
			key:      "cryptoKeyVersions",
			input:    "projects/a/locations/b/keyRings/r/cryptoKeys/key/cryptoKeyVersions/7",
			expected: "7",
		},
		{
			name:     "ExtractCryptoKeyVersion: Missing version segment",
			key:      "cryptoKeyVersions",
			input:    "projects/p/locations/l/keyRings/r/cryptoKeys/key",
			expected: "",
		},
		{
			name:     "ExtractCryptoKeyVersion: Empty input",
			key:      "cryptoKeyVersions",
			input:    "",
			expected: "",
		},
		{
			name:     "ExtractCryptoKeyVersion: Malformed string",
			key:      "cryptoKeyVersions",
			input:    "cryptoKeyVersions",
			expected: "",
		},

		// ExtractZone cases (as ExtractResourcePart)
		{
			name:     "Valid input with zone",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-a/disks/integration-test-instance",
			expected: "us-central1-a",
			key:      "zones",
		},
		{
			name:     "Valid input with different zone",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/europe-west1-b/disks/integration-test-instance",
			expected: "europe-west1-b",
			key:      "zones",
		},
		{
			name:     "Valid input shortened",
			input:    "zones/zone/disks/disk",
			expected: "zone",
			key:      "zones",
		},
		{
			name:     "Input without zones",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default",
			expected: "",
			key:      "zones",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
			key:      "zones",
		},
		{
			name:     "Malformed input",
			input:    "invalid-string-without-zones",
			expected: "",
			key:      "zones",
		},

		// ExtractRegions cases (as ExtractResourcePart)
		{
			name:     "Valid input with region",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default",
			expected: "us-central1",
			key:      "regions",
		},
		{
			name:     "Valid input with different region",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/regions/europe-west1/subnetworks/default",
			expected: "europe-west1",
			key:      "regions",
		},
		{
			name:     "Valid input shortened",
			input:    "regions/region/subnetworks/subnetwork",
			expected: "region",
			key:      "regions",
		},
		{
			name:     "Input without regions",
			input:    "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-a/instances/instance-1",
			expected: "",
			key:      "regions",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
			key:      "regions",
		},
		{
			name:     "Malformed input",
			input:    "invalid-string-without-regions",
			expected: "",
			key:      "regions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.ExtractPathParam(tt.key, tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPathParam(%q, %q) = %q; want %q", tt.input, tt.key, result, tt.expected)
			}
		})
	}
}

func TestShortenSelfLink(t *testing.T) {
	tests := []struct {
		name     string
		selfLink string
		expected string
	}{
		{
			name:     "Valid input",
			selfLink: "https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-c/instanceGroupManagers/test-igm",
			expected: "zones/us-central1-c/instanceGroupManagers/test-igm",
		},
		{
			name:     "Empty input",
			selfLink: "",
			expected: "",
		},
		{
			name:     "Malformed input",
			selfLink: "invalid/selfLink/format",
			expected: "invalid/selfLink/format",
		},
		{
			name:     "Short input",
			selfLink: "https://www.googleapis.com/compute/v1/projects/test-project",
			expected: "compute/v1/projects/test-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.ShortenSelfLink(tt.selfLink)
			if result != tt.expected {
				t.Errorf("ShortenSelfLink(%q) = %q, want %q", tt.selfLink, result, tt.expected)
			}
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keys     []string
		expected []string
	}{
		{
			name:     "single key present",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring",
			keys:     []string{"locations"},
			expected: []string{"us-central1"},
		},
		{
			name:     "multiple keys, both present",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key",
			keys:     []string{"keyRings", "cryptoKeys"},
			expected: []string{"my-key", "my-ring"},
		},
		{
			name:     "multiple keys, one missing",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring",
			keys:     []string{"keyRings", "cryptoKeys"},
			expected: []string{"my-ring"},
		},
		{
			name:     "all keys missing",
			input:    "projects/proj/locations/us-central1",
			keys:     []string{"foo", "bar"},
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			keys:     []string{"locations"},
			expected: []string{},
		},
		{
			name:     "empty keys",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring",
			keys:     []string{},
			expected: []string{},
		},
		{
			name:     "key at end, no value",
			input:    "projects/proj/locations",
			keys:     []string{"locations"},
			expected: []string{},
		},
		{
			name:     "multiple keys, both present, reverse order",
			input:    "projects/proj/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key",
			keys:     []string{"locations", "cryptoKeys"},
			expected: []string{"my-key", "us-central1"},
		},
		{
			name:     "default",
			input:    "default",
			keys:     []string{"subnetworks"},
			expected: []string{"default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gcpshared.ExtractPathParams(tt.input, tt.keys...)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractPathParams(%q, %v) returned %d results, want %d", tt.input, tt.keys, len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ExtractPathParams(%q, %v)[%d] = %q; want %q", tt.input, tt.keys, i, result[i], tt.expected[i])
				}
			}
		})
	}
}
