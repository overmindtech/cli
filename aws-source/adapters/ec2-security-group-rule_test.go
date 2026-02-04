package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestSecurityGroupRuleInputMapperGet(t *testing.T) {
	input, err := securityGroupRuleInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.SecurityGroupRuleIds) != 1 {
		t.Fatalf("expected 1 SecurityGroupRule ID, got %v", len(input.SecurityGroupRuleIds))
	}

	if input.SecurityGroupRuleIds[0] != "bar" {
		t.Errorf("expected SecurityGroupRule ID to be bar, got %v", input.SecurityGroupRuleIds[0])
	}
}

func TestSecurityGroupRuleInputMapperList(t *testing.T) {
	input, err := securityGroupRuleInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.SecurityGroupRuleIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestSecurityGroupRuleOutputMapper(t *testing.T) {
	output := &ec2.DescribeSecurityGroupRulesOutput{
		SecurityGroupRules: []types.SecurityGroupRule{
			{
				SecurityGroupRuleId: PtrString("sgr-0b0e42d1431e832bd"),
				GroupId:             PtrString("sg-0814766e46f201c22"),
				GroupOwnerId:        PtrString("052392120703"),
				IsEgress:            PtrBool(false),
				IpProtocol:          PtrString("tcp"),
				FromPort:            PtrInt32(2049),
				ToPort:              PtrInt32(2049),
				ReferencedGroupInfo: &types.ReferencedSecurityGroup{
					GroupId: PtrString("sg-09371b4a54fe7ab38"),
					UserId:  PtrString("052392120703"),
				},
				Description: PtrString("Created by the LIW for EFS at 2022-12-16T19:14:27.033Z"),
				Tags:        []types.Tag{},
			},
			{
				SecurityGroupRuleId: PtrString("sgr-04b583a90b4fa4ada"),
				GroupId:             PtrString("sg-09371b4a54fe7ab38"),
				GroupOwnerId:        PtrString("052392120703"),
				IsEgress:            PtrBool(true),
				IpProtocol:          PtrString("tcp"),
				FromPort:            PtrInt32(2049),
				ToPort:              PtrInt32(2049),
				ReferencedGroupInfo: &types.ReferencedSecurityGroup{
					GroupId: PtrString("sg-0814766e46f201c22"),
					UserId:  PtrString("052392120703"),
				},
				Description: PtrString("Created by the LIW for EFS at 2022-12-16T19:14:27.349Z"),
				Tags:        []types.Tag{},
			},
		},
	}

	items, err := securityGroupRuleOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := QueryTests{
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg-0814766e46f201c22",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg-09371b4a54fe7ab38",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2SecurityGroupRuleAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2SecurityGroupRuleAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
