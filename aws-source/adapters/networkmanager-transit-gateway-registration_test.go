package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestTransitGatewayRegistrationOutputMapper(t *testing.T) {
	scope := "123456789012.eu-west-2"
	tests := []struct {
		name         string
		out          networkmanager.GetTransitGatewayRegistrationsOutput
		expectedAttr string
		tests        adapterhelpers.QueryTests
	}{
		{
			name: "ok",
			out: networkmanager.GetTransitGatewayRegistrationsOutput{
				TransitGatewayRegistrations: []types.TransitGatewayRegistration{
					{
						GlobalNetworkId:   adapterhelpers.PtrString("default"),
						TransitGatewayArn: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234"),
						State: &types.TransitGatewayRegistrationStateReason{
							Code: types.TransitGatewayRegistrationStateAvailable,
						},
					},
				},
			},
			expectedAttr: "default|arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234",
			tests: adapterhelpers.QueryTests{
				{
					ExpectedType:   "networkmanager-global-network",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   "ec2-transit-gateway",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234",
					ExpectedScope:  "123456789012.us-west-2",
				},
			},
		},
		{
			name: "ok, deleting",
			out: networkmanager.GetTransitGatewayRegistrationsOutput{
				TransitGatewayRegistrations: []types.TransitGatewayRegistration{
					{
						GlobalNetworkId:   adapterhelpers.PtrString("default"),
						TransitGatewayArn: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234"),
						State: &types.TransitGatewayRegistrationStateReason{
							Code: types.TransitGatewayRegistrationStateDeleting,
						},
					},
				},
			},
			expectedAttr: "default|arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234",
			tests: adapterhelpers.QueryTests{
				{
					ExpectedType:   "networkmanager-global-network",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   "ec2-transit-gateway",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234",
					ExpectedScope:  "123456789012.us-west-2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := transitGatewayRegistrationOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetTransitGatewayRegistrationsInput{}, &tt.out)
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

			if item.UniqueAttributeValue() != tt.expectedAttr {
				t.Fatalf("want %s, got %s", tt.expectedAttr, item.UniqueAttributeValue())
			}

			tt.tests.Execute(t, item)
		})
	}
}
