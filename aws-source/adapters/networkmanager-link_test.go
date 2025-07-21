package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestLinkOutputMapper(t *testing.T) {
	output := networkmanager.GetLinksOutput{
		Links: []types.Link{
			{
				LinkId:          adapterhelpers.PtrString("link-1"),
				GlobalNetworkId: adapterhelpers.PtrString("default"),
				SiteId:          adapterhelpers.PtrString("site-1"),
				LinkArn:         adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:link/link-1"),
			},
		},
	}
	scope := "123456789012.eu-west-2"
	items, err := linkOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetLinksInput{}, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// Ensure unique attribute
	err = item.Validate()

	if err != nil {
		t.Error(err)
	}

	if item.UniqueAttributeValue() != "default|link-1" {
		t.Fatalf("expected default|link-1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-site",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|site-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-network-resource-relationship",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|arn:aws:networkmanager:us-west-2:123456789012:link/link-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link-association",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|link|link-1",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestLinkInputMapperSearch(t *testing.T) {
	adapter := NewNetworkManagerLinkAdapter(&networkmanager.Client{}, "123456789012")

	tests := []struct {
		name          string
		query         string
		expectedInput *networkmanager.GetLinksInput
		expectError   bool
	}{
		{
			name:  "Valid networkmanager-link ARN",
			query: "arn:aws:networkmanager::123456789012:link/global-network-01231231231231231/link-11112222aaaabbbb1",
			expectedInput: &networkmanager.GetLinksInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-01231231231231231"),
				LinkIds:         []string{"link-11112222aaaabbbb1"},
			},
			expectError: false,
		},
		{
			name:  "Global Network ID only",
			query: "global-network-123456789",
			expectedInput: &networkmanager.GetLinksInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-123456789"),
			},
			expectError: false,
		},
		{
			name:  "Global Network ID and Site ID",
			query: "global-network-123456789|site-987654321",
			expectedInput: &networkmanager.GetLinksInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-123456789"),
				SiteId:          adapterhelpers.PtrString("site-987654321"),
			},
			expectError: false,
		},
		{
			name:        "Invalid ARN - wrong service",
			query:       "arn:aws:ec2::123456789012:instance/i-1234567890abcdef0",
			expectError: true,
		},
		{
			name:        "Invalid ARN - wrong resource type",
			query:       "arn:aws:networkmanager::123456789012:device/global-network-01231231231231231/device-444555aaabbb11223",
			expectError: true,
		},
		{
			name:        "Invalid ARN - malformed resource",
			query:       "arn:aws:networkmanager::123456789012:link/invalid-format",
			expectError: true,
		},
		{
			name:        "Invalid ARN - wrong prefixes",
			query:       "arn:aws:networkmanager::123456789012:link/global-network-123/invalid-prefix-123",
			expectError: true,
		},
		{
			name:        "Invalid query - too many sections",
			query:       "section1|section2|section3",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := adapter.InputMapperSearch(context.Background(), &networkmanager.Client{}, "123456789012.us-east-1", tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for query %s, but got none", tt.query)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for query %s: %v", tt.query, err)
				return
			}

			if input == nil {
				t.Errorf("Expected input but got nil for query %s", tt.query)
				return
			}

			// Compare GlobalNetworkId
			if (input.GlobalNetworkId == nil) != (tt.expectedInput.GlobalNetworkId == nil) {
				t.Errorf("GlobalNetworkId nil mismatch for query %s", tt.query)
				return
			}
			if input.GlobalNetworkId != nil && tt.expectedInput.GlobalNetworkId != nil {
				if *input.GlobalNetworkId != *tt.expectedInput.GlobalNetworkId {
					t.Errorf("Expected GlobalNetworkId %s, got %s for query %s",
						*tt.expectedInput.GlobalNetworkId, *input.GlobalNetworkId, tt.query)
				}
			}

			// Compare SiteId
			if (input.SiteId == nil) != (tt.expectedInput.SiteId == nil) {
				t.Errorf("SiteId nil mismatch for query %s", tt.query)
				return
			}
			if input.SiteId != nil && tt.expectedInput.SiteId != nil {
				if *input.SiteId != *tt.expectedInput.SiteId {
					t.Errorf("Expected SiteId %s, got %s for query %s",
						*tt.expectedInput.SiteId, *input.SiteId, tt.query)
				}
			}

			// Compare LinkIds
			if len(input.LinkIds) != len(tt.expectedInput.LinkIds) {
				t.Errorf("Expected %d LinkIds, got %d for query %s",
					len(tt.expectedInput.LinkIds), len(input.LinkIds), tt.query)
				return
			}
			for i, linkId := range input.LinkIds {
				if linkId != tt.expectedInput.LinkIds[i] {
					t.Errorf("Expected LinkId %s, got %s at index %d for query %s",
						tt.expectedInput.LinkIds[i], linkId, i, tt.query)
				}
			}
		})
	}
}
