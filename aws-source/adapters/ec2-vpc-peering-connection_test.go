package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestVpcPeeringConnectionOutputMapper(t *testing.T) {
	output := &ec2.DescribeVpcPeeringConnectionsOutput{
		VpcPeeringConnections: []types.VpcPeeringConnection{
			{
				VpcPeeringConnectionId: adapterhelpers.PtrString("pcx-1234567890"),
				Status: &types.VpcPeeringConnectionStateReason{
					Code:    types.VpcPeeringConnectionStateReasonCodeActive, // health
					Message: adapterhelpers.PtrString("message"),
				},
				AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
					CidrBlock: adapterhelpers.PtrString("10.0.0.1/24"),
					CidrBlockSet: []types.CidrBlock{
						{
							CidrBlock: adapterhelpers.PtrString("10.0.2.1/24"),
						},
					},
					Ipv6CidrBlockSet: []types.Ipv6CidrBlock{
						{
							Ipv6CidrBlock: adapterhelpers.PtrString("::/64"),
						},
					},
					OwnerId: adapterhelpers.PtrString("123456789012"),
					Region:  adapterhelpers.PtrString("eu-west-2"),      // link
					VpcId:   adapterhelpers.PtrString("vpc-1234567890"), // link
					PeeringOptions: &types.VpcPeeringConnectionOptionsDescription{
						AllowDnsResolutionFromRemoteVpc: adapterhelpers.PtrBool(true),
					},
				},
				RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
					CidrBlock: adapterhelpers.PtrString("10.0.0.1/24"),
					CidrBlockSet: []types.CidrBlock{
						{
							CidrBlock: adapterhelpers.PtrString("10.0.2.1/24"),
						},
					},
					Ipv6CidrBlockSet: []types.Ipv6CidrBlock{
						{
							Ipv6CidrBlock: adapterhelpers.PtrString("::/64"),
						},
					},
					OwnerId: adapterhelpers.PtrString("987654321098"),
					PeeringOptions: &types.VpcPeeringConnectionOptionsDescription{
						AllowDnsResolutionFromRemoteVpc: adapterhelpers.PtrBool(true),
					},
					Region: adapterhelpers.PtrString("eu-west-5"),      // link
					VpcId:  adapterhelpers.PtrString("vpc-9887654321"), // link
				},
			},
		},
	}

	items, err := vpcPeeringConnectionOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
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

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-1234567890",
			ExpectedScope:  "123456789012.eu-west-2",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-9887654321",
			ExpectedScope:  "987654321098.eu-west-5",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2VpcPeeringConnectionAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2VpcPeeringConnectionAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
