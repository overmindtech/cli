package sources

import (
	"testing"

	aws "github.com/overmindtech/cli/sources/aws/shared"
	gcp "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestItemTypeReadableFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    shared.ItemType
		expected string
	}{
		{
			name:     "Three parts input",
			input:    shared.NewItemType(gcp.GCP, gcp.Compute, gcp.Instance),
			expected: "GCP Compute Instance",
		},
		{
			name:     "Three parts input",
			input:    shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI),
			expected: "AWS Api Gateway Rest Api",
			// Note that this is only testing the fallback rendering,
			// adapter implementors will have to supply a custom descriptive name,
			// like "Amazon API Gateway REST API" in the `AdapterMetadata`.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.input.Readable()
			if actual != tt.expected {
				t.Errorf("readableFormat(%q) = %q; expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}
