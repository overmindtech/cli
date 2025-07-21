package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestConnectionOutputMapper(t *testing.T) {
	output := networkmanager.GetConnectionsOutput{
		Connections: []types.Connection{
			{
				GlobalNetworkId:   adapterhelpers.PtrString("default"),
				ConnectionId:      adapterhelpers.PtrString("conn-1"),
				DeviceId:          adapterhelpers.PtrString("dvc-1"),
				ConnectedDeviceId: adapterhelpers.PtrString("dvc-2"),
				LinkId:            adapterhelpers.PtrString("link-1"),
				ConnectedLinkId:   adapterhelpers.PtrString("link-2"),
			},
		},
	}
	scope := "123456789012.eu-west-2"
	items, err := connectionOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetConnectionsInput{}, &output)

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

	if item.UniqueAttributeValue() != "default|conn-1" {
		t.Fatalf("expected default|conn-1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-device",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|dvc-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-device",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|dvc-2",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|link-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|link-2",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestConnectionInputMapperSearch(t *testing.T) {
	adapter := NewNetworkManagerConnectionAdapter(&networkmanager.Client{}, "123456789012")

	tests := []struct {
		name          string
		query         string
		expectedInput *networkmanager.GetConnectionsInput
		expectError   bool
	}{
		{
			name:  "Valid networkmanager-connection ARN",
			query: "arn:aws:networkmanager::123456789012:device/global-network-0d47f6t230mz46dy4/connection-07f6fd08867abc123",
			expectedInput: &networkmanager.GetConnectionsInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-0d47f6t230mz46dy4"),
				ConnectionIds:   []string{"connection-07f6fd08867abc123"},
			},
			expectError: false,
		},
		{
			name:  "Valid networkmanager-device ARN",
			query: "arn:aws:networkmanager::123456789012:device/global-network-01231231231231231/device-07f6fd08867abc123",
			expectedInput: &networkmanager.GetConnectionsInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-01231231231231231"),
				DeviceId:        adapterhelpers.PtrString("device-07f6fd08867abc123"),
			},
			expectError: false,
		},
		{
			name:  "Global Network ID only",
			query: "global-network-123456789",
			expectedInput: &networkmanager.GetConnectionsInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-123456789"),
			},
			expectError: false,
		},
		{
			name:  "Global Network ID and Device ID",
			query: "global-network-123456789|device-987654321",
			expectedInput: &networkmanager.GetConnectionsInput{
				GlobalNetworkId: adapterhelpers.PtrString("global-network-123456789"),
				DeviceId:        adapterhelpers.PtrString("device-987654321"),
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
			query:       "arn:aws:networkmanager::123456789012:site/global-network-01231231231231231/site-444555aaabbb11223",
			expectError: true,
		},
		{
			name:        "Invalid connection ARN - malformed resource",
			query:       "arn:aws:networkmanager::123456789012:device/invalid-format",
			expectError: true,
		},
		{
			name:        "Invalid device ARN - malformed resource",
			query:       "arn:aws:networkmanager::123456789012:device/global-network-123/invalid-prefix-123",
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

			// Compare DeviceId
			if (input.DeviceId == nil) != (tt.expectedInput.DeviceId == nil) {
				t.Errorf("DeviceId nil mismatch for query %s", tt.query)
				return
			}
			if input.DeviceId != nil && tt.expectedInput.DeviceId != nil {
				if *input.DeviceId != *tt.expectedInput.DeviceId {
					t.Errorf("Expected DeviceId %s, got %s for query %s",
						*tt.expectedInput.DeviceId, *input.DeviceId, tt.query)
				}
			}

			// Compare ConnectionIds
			if len(input.ConnectionIds) != len(tt.expectedInput.ConnectionIds) {
				t.Errorf("Expected %d ConnectionIds, got %d for query %s",
					len(tt.expectedInput.ConnectionIds), len(input.ConnectionIds), tt.query)
				return
			}
			for i, connectionId := range input.ConnectionIds {
				if connectionId != tt.expectedInput.ConnectionIds[i] {
					t.Errorf("Expected ConnectionId %s, got %s at index %d for query %s",
						tt.expectedInput.ConnectionIds[i], connectionId, i, tt.query)
				}
			}
		})
	}
}
