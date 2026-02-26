package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestVpcInputMapperGet(t *testing.T) {
	input, err := vpcInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.VpcIds) != 1 {
		t.Fatalf("expected 1 Vpc ID, got %v", len(input.VpcIds))
	}

	if input.VpcIds[0] != "bar" {
		t.Errorf("expected Vpc ID to be bar, got %v", input.VpcIds[0])
	}
}

func TestVpcInputMapperList(t *testing.T) {
	input, err := vpcInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.VpcIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestVpcOutputMapper(t *testing.T) {
	output := &ec2.DescribeVpcsOutput{
		Vpcs: []types.Vpc{
			{
				CidrBlock:       new("172.31.0.0/16"),
				DhcpOptionsId:   new("dopt-0959b838bf4a4c7b8"),
				State:           types.VpcStateAvailable,
				VpcId:           new("vpc-0d7892e00e573e701"),
				OwnerId:         new("052392120703"),
				InstanceTenancy: types.TenancyDefault,
				CidrBlockAssociationSet: []types.VpcCidrBlockAssociation{
					{
						AssociationId: new("vpc-cidr-assoc-0b77866f37f500af6"),
						CidrBlock:     new("172.31.0.0/16"),
						CidrBlockState: &types.VpcCidrBlockState{
							State: types.VpcCidrBlockStateCodeAssociated,
						},
					},
				},
				IsDefault: new(false),
				Tags: []types.Tag{
					{
						Key:   new("aws:cloudformation:logical-id"),
						Value: new("VPC"),
					},
					{
						Key:   new("aws:cloudformation:stack-id"),
						Value: new("arn:aws:cloudformation:eu-west-2:052392120703:stack/StackSet-AWSControlTowerBP-VPC-ACCOUNT-FACTORY-V1-8c2a9348-a30c-4ac3-94c2-8279157c9243/ccde3240-7afa-11ed-81ff-02845d4c2702"),
					},
					{
						Key:   new("aws:cloudformation:stack-name"),
						Value: new("StackSet-AWSControlTowerBP-VPC-ACCOUNT-FACTORY-V1-8c2a9348-a30c-4ac3-94c2-8279157c9243"),
					},
					{
						Key:   new("Name"),
						Value: new("aws-controltower-VPC"),
					},
				},
			},
		},
	}

	items, err := vpcOutputMapper(context.Background(), nil, "foo", nil, output)

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
}

func TestNewEC2VpcAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2VpcAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
