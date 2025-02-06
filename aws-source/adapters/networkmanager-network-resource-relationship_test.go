package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestNetworkResourceRelationshipOutputMapper(t *testing.T) {
	scope := "123456789012.eu-west-2"
	tests := []struct {
		name   string
		input  networkmanager.GetNetworkResourceRelationshipsInput
		output networkmanager.GetNetworkResourceRelationshipsOutput
		tests  []adapterhelpers.QueryTests
	}{
		{
			name: "ok, one entity",
			input: networkmanager.GetNetworkResourceRelationshipsInput{
				GlobalNetworkId: adapterhelpers.PtrString("default"),
			},
			output: networkmanager.GetNetworkResourceRelationshipsOutput{
				Relationships: []types.Relationship{
					// connection, device
					{
						From: adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:connection/conn-1"),
						To:   adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:device/d-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:connection/conn-1"),
						From: adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:device/d-1"),
					},
					// link, site
					{
						From: adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:link/link-1"),
						To:   adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:site/site-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:link/link-1"),
						From: adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:site/site-1"),
					},
					// directconnect-connection, directconnect-direct-connect-gateway
					{
						From: adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:connection/dxconn-1"),
						To:   adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:direct-connect-gateway/gw-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:connection/dxconn-1"),
						From: adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:direct-connect-gateway/gw-1"),
					},
					// directconnect-virtual-interface, ec2-customer-gateway
					{
						From: adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:virtual-interface/vif-1"),
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:customer-gateway/gw-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:directconnect:us-west-2:123456789012:virtual-interface/vif-1"),
						From: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:customer-gateway/gw-1"),
					},
					// ec2-transit-gateway, ec2-transit-gateway-attachment
					{
						From: adapterhelpers.PtrString("arn:aws:ec2:us-east-2:986543144159:transit-gateway/tgw-06910e97a1fbdf66a"),
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway-attachment/tgwa-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-east-2:986543144159:transit-gateway/tgw-06910e97a1fbdf66a"),
						From: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway-attachment/tgwa-1"),
					},
					// ec2-transit-gateway-route-table, ec2-transit-gateway-connect-peer
					{
						From: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway-connect-peer/tgw-cnp-1"),
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-east-2:986543144159:transit-gateway-route-table/tgw-rtb-043b7b4c0db1e4833"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:transit-gateway-connect-peer/tgw-cnp-1"),
						From: adapterhelpers.PtrString("arn:aws:ec2:us-east-2:986543144159:transit-gateway-route-table/tgw-rtb-043b7b4c0db1e4833"),
					},
					// connection, ec2-vpn-connection
					{
						From: adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:connection/conn-1"),
						To:   adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:vpn-connection/conn-1"),
					},
					{
						To:   adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:connection/conn-1"),
						From: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:vpn-connection/conn-1"),
					},
				},
			},
			tests: []adapterhelpers.QueryTests{
				// connection to device
				{
					{
						ExpectedType:   "networkmanager-device",
						ExpectedMethod: sdp.QueryMethod_SEARCH,
						ExpectedQuery:  "default|d-1",
						ExpectedScope:  scope,
					},
				},
				// device to connection
				{
					{
						ExpectedType:   "networkmanager-connection",
						ExpectedMethod: sdp.QueryMethod_SEARCH,
						ExpectedQuery:  "default|conn-1",
						ExpectedScope:  scope,
					},
				},
				// link to site
				{
					{
						ExpectedType:   "networkmanager-site",
						ExpectedMethod: sdp.QueryMethod_SEARCH,
						ExpectedQuery:  "default|site-1",
						ExpectedScope:  scope,
					},
				},
				// site to link
				{
					{
						ExpectedType:   "networkmanager-link",
						ExpectedMethod: sdp.QueryMethod_SEARCH,
						ExpectedQuery:  "default|link-1",
						ExpectedScope:  scope,
					},
				},
				// directconnect-connection to directconnect-direct-connect-gateway
				{
					{
						ExpectedType:   "directconnect-direct-connect-gateway",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "gw-1",
						ExpectedScope:  scope,
					},
				},
				// directconnect-direct-connect-gateway to directconnect-connection
				{
					{
						ExpectedType:   "directconnect-connection",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "dxconn-1",
						ExpectedScope:  scope,
					},
				},
				// directconnect-virtual-interface to ec2-customer-gateway
				{
					{
						ExpectedType:   "ec2-customer-gateway",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "gw-1",
						ExpectedScope:  scope,
					},
				},
				// ec2-customer-gateway to directconnect-virtual-interface
				{
					{
						ExpectedType:   "directconnect-virtual-interface",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "vif-1",
						ExpectedScope:  scope,
					},
				},
				// ec2-transit-gateway to ec2-transit-gateway-attachment
				{
					{
						ExpectedType:   "ec2-transit-gateway-attachment",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "tgwa-1",
						ExpectedScope:  scope,
					},
				},
				// ec2-transit-gateway-attachment to ec2-transit-gateway
				{
					{
						ExpectedType:   "ec2-transit-gateway",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "tgw-06910e97a1fbdf66a",
						ExpectedScope:  scope,
					},
				},
				// ec2-transit-gateway-connect-peer to ec2-transit-gateway-route-table
				{
					{
						ExpectedType:   "ec2-transit-gateway-route-table",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "tgw-rtb-043b7b4c0db1e4833",
						ExpectedScope:  scope,
					},
				},
				// ec2-transit-gateway-route-table to ec2-transit-gateway-connect-peer
				{
					{
						ExpectedType:   "ec2-transit-gateway-connect-peer",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "tgw-cnp-1",
						ExpectedScope:  scope,
					},
				},
				// connection to ec2-vpn-connection
				{
					{
						ExpectedType:   "ec2-vpn-connection",
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "conn-1",
						ExpectedScope:  scope,
					},
				},
				// ec2-vpn-connection to connection
				{
					{
						ExpectedType:   "networkmanager-connection",
						ExpectedMethod: sdp.QueryMethod_SEARCH,
						ExpectedQuery:  "default|conn-1",
						ExpectedScope:  scope,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := networkResourceRelationshipOutputMapper(context.Background(), &networkmanager.Client{}, scope, &tt.input, &tt.output)
			if err != nil {
				t.Error(err)
			}
			for i := range items {
				if err := items[i].Validate(); err != nil {
					t.Error(err)
				}
				tt.tests[i].Execute(t, items[i])
			}

		})
	}
}
