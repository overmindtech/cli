package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestSiteOutputMapper(t *testing.T) {
	output := networkmanager.GetSitesOutput{
		Sites: []types.Site{
			{
				SiteId:          adapterhelpers.PtrString("site1"),
				GlobalNetworkId: adapterhelpers.PtrString("default"),
			},
		},
	}
	scope := "123456789012.eu-west-2"
	items, err := siteOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetSitesInput{}, &output)

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

	if item.UniqueAttributeValue() != "default|site1" {
		t.Fatalf("expected default|site1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|site1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-device",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|site1",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestSiteInputMapperSearch(t *testing.T) {
	adapter := NewNetworkManagerSiteAdapter(&networkmanager.Client{}, "123456789012")

	tests := []struct {
		name          string
		query         string
		expectedInput *networkmanager.GetSitesInput
		expectError   bool
	}{
		{
			name:  "Valid networkmanager-site ARN",
			query: "arn:aws:networkmanager::123456789012:site/global-network-01231231231231231/site-444555aaabbb11223",
			expectedInput: &networkmanager.GetSitesInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-01231231231231231"),
				SiteIds:         []string{"site-444555aaabbb11223"},
			},
			expectError: false,
		},
		{
			name:  "Global Network ID (backward compatibility)",
			query: "global-network-123456789",
			expectedInput: &networkmanager.GetSitesInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-123456789"),
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
			query:       "arn:aws:networkmanager::123456789012:site/invalid-format",
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

			// Compare SiteIds
			if len(input.SiteIds) != len(tt.expectedInput.SiteIds) {
				t.Errorf("Expected %d SiteIds, got %d for query %s",
					len(tt.expectedInput.SiteIds), len(input.SiteIds), tt.query)
				return
			}
			for i, siteId := range input.SiteIds {
				if siteId != tt.expectedInput.SiteIds[i] {
					t.Errorf("Expected SiteId %s, got %s at index %d for query %s",
						tt.expectedInput.SiteIds[i], siteId, i, tt.query)
				}
			}
		})
	}
}
