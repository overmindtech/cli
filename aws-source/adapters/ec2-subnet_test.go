package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestSubnetInputMapperGet(t *testing.T) {
	input, err := subnetInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.SubnetIds) != 1 {
		t.Fatalf("expected 1 Subnet ID, got %v", len(input.SubnetIds))
	}

	if input.SubnetIds[0] != "bar" {
		t.Errorf("expected Subnet ID to be bar, got %v", input.SubnetIds[0])
	}
}

func TestSubnetInputMapperList(t *testing.T) {
	input, err := subnetInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.SubnetIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestSubnetOutputMapper(t *testing.T) {
	output := &ec2.DescribeSubnetsOutput{
		Subnets: []types.Subnet{
			{
				AvailabilityZone:            new("eu-west-2c"),
				AvailabilityZoneId:          new("euw2-az1"),
				AvailableIpAddressCount:     new(int32(4091)),
				CidrBlock:                   new("172.31.80.0/20"),
				DefaultForAz:                new(false),
				MapPublicIpOnLaunch:         new(false),
				MapCustomerOwnedIpOnLaunch:  new(false),
				State:                       types.SubnetStateAvailable,
				SubnetId:                    new("subnet-0450a637af9984235"),
				VpcId:                       new("vpc-0d7892e00e573e701"),
				OwnerId:                     new("052392120703"),
				AssignIpv6AddressOnCreation: new(false),
				Ipv6CidrBlockAssociationSet: []types.SubnetIpv6CidrBlockAssociation{
					{
						AssociationId: new("id-1234"),
						Ipv6CidrBlock: new("something"),
						Ipv6CidrBlockState: &types.SubnetCidrBlockState{
							State:         types.SubnetCidrBlockStateCodeAssociated,
							StatusMessage: new("something here"),
						},
					},
				},
				Tags:        []types.Tag{},
				SubnetArn:   new("arn:aws:ec2:eu-west-2:052392120703:subnet/subnet-0450a637af9984235"),
				EnableDns64: new(false),
				Ipv6Native:  new(false),
				PrivateDnsNameOptionsOnLaunch: &types.PrivateDnsNameOptionsOnLaunch{
					HostnameType:                    types.HostnameTypeIpName,
					EnableResourceNameDnsARecord:    new(false),
					EnableResourceNameDnsAAAARecord: new(false),
				},
			},
		},
	}

	items, err := subnetOutputMapper(context.Background(), nil, "foo", nil, output)

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
	tests := QueryTests{
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2SubnetAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2SubnetAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
