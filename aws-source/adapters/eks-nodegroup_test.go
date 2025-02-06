package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

var NodeGroupClient = EKSTestClient{
	DescribeNodegroupOutput: &eks.DescribeNodegroupOutput{
		Nodegroup: &types.Nodegroup{
			NodegroupName:  adapterhelpers.PtrString("default-2022122213523169820000001f"),
			NodegroupArn:   adapterhelpers.PtrString("arn:aws:eks:eu-west-2:801795385023:nodegroup/dylan/default-2022122213523169820000001f/98c29d0d-b22a-aaa3-445e-ebf71d43f67c"),
			ClusterName:    adapterhelpers.PtrString("dylan"),
			Version:        adapterhelpers.PtrString("1.24"),
			ReleaseVersion: adapterhelpers.PtrString("1.24.7-20221112"),
			CreatedAt:      adapterhelpers.PtrTime(time.Now()),
			ModifiedAt:     adapterhelpers.PtrTime(time.Now()),
			Status:         types.NodegroupStatusActive,
			CapacityType:   types.CapacityTypesOnDemand,
			DiskSize:       adapterhelpers.PtrInt32(100),
			RemoteAccess: &types.RemoteAccessConfig{
				Ec2SshKey: adapterhelpers.PtrString("key"), // link
				SourceSecurityGroups: []string{
					"sg1", // link
				},
			},
			ScalingConfig: &types.NodegroupScalingConfig{
				MinSize:     adapterhelpers.PtrInt32(1),
				MaxSize:     adapterhelpers.PtrInt32(3),
				DesiredSize: adapterhelpers.PtrInt32(1),
			},
			InstanceTypes: []string{
				"T3large",
			},
			Subnets: []string{
				"subnet0d1fabfe6794b5543", // link
			},
			AmiType:  types.AMITypesAl2Arm64,
			NodeRole: adapterhelpers.PtrString("arn:aws:iam::801795385023:role/default-eks-node-group-20221222134106992000000003"),
			Labels:   map[string]string{},
			Taints: []types.Taint{
				{
					Effect: types.TaintEffectNoSchedule,
					Key:    adapterhelpers.PtrString("key"),
					Value:  adapterhelpers.PtrString("value"),
				},
			},
			Resources: &types.NodegroupResources{
				AutoScalingGroups: []types.AutoScalingGroup{
					{
						Name: adapterhelpers.PtrString("eks-default-2022122213523169820000001f-98c29d0d-b22a-aaa3-445e-ebf71d43f67c"), // link
					},
				},
				RemoteAccessSecurityGroup: adapterhelpers.PtrString("sg2"), // link
			},
			Health: &types.NodegroupHealth{
				Issues: []types.Issue{},
			},
			UpdateConfig: &types.NodegroupUpdateConfig{
				MaxUnavailablePercentage: adapterhelpers.PtrInt32(33),
			},
			LaunchTemplate: &types.LaunchTemplateSpecification{
				Name:    adapterhelpers.PtrString("default-2022122213523100410000001d"), // link
				Version: adapterhelpers.PtrString("1"),
				Id:      adapterhelpers.PtrString("lt-097e994ce7e14fcdc"),
			},
			Tags: map[string]string{},
		},
	},
}

func TestNodegroupGetFunc(t *testing.T) {
	item, err := nodegroupGetFunc(context.Background(), NodeGroupClient, "foo", &eks.DescribeNodegroupInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-key-pair",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "key",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg1",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet0d1fabfe6794b5543",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "eks-default-2022122213523169820000001f-98c29d0d-b22a-aaa3-445e-ebf71d43f67c",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg2",
			ExpectedScope:  item.GetScope(),
		},
		{
			ExpectedType:   "ec2-launch-template",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "lt-097e994ce7e14fcdc",
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)
}

func TestNewEKSNodegroupAdapter(t *testing.T) {
	client, account, region := eksGetAutoConfig(t)

	adapter := NewEKSNodegroupAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
