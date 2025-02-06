package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestAutoScalingGroupOutputMapper(t *testing.T) {
	t.Parallel()

	output := autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []types.AutoScalingGroup{
			{
				AutoScalingGroupName: adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
				AutoScalingGroupARN:  adapterhelpers.PtrString("arn:aws:autoscaling:eu-west-2:944651592624:autoScalingGroup:1cbb0e22-818f-4d8b-8662-77f73d3713ca:autoScalingGroupName/eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
				MixedInstancesPolicy: &types.MixedInstancesPolicy{
					LaunchTemplate: &types.LaunchTemplate{
						LaunchTemplateSpecification: &types.LaunchTemplateSpecification{
							LaunchTemplateId:   adapterhelpers.PtrString("lt-0174ff2b8909d0c75"), // link
							LaunchTemplateName: adapterhelpers.PtrString("eks-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
							Version:            adapterhelpers.PtrString("1"),
						},
						Overrides: []types.LaunchTemplateOverrides{
							{
								InstanceType: adapterhelpers.PtrString("t3.large"),
							},
						},
					},
					InstancesDistribution: &types.InstancesDistribution{
						OnDemandAllocationStrategy:          adapterhelpers.PtrString("prioritized"),
						OnDemandBaseCapacity:                adapterhelpers.PtrInt32(0),
						OnDemandPercentageAboveBaseCapacity: adapterhelpers.PtrInt32(100),
						SpotAllocationStrategy:              adapterhelpers.PtrString("lowest-price"),
						SpotInstancePools:                   adapterhelpers.PtrInt32(2),
					},
				},
				MinSize:         adapterhelpers.PtrInt32(1),
				MaxSize:         adapterhelpers.PtrInt32(3),
				DesiredCapacity: adapterhelpers.PtrInt32(1),
				DefaultCooldown: adapterhelpers.PtrInt32(300),
				AvailabilityZones: []string{ // link
					"eu-west-2c",
					"eu-west-2a",
					"eu-west-2b",
				},
				LoadBalancerNames: []string{}, // Ignored, classic load balancer
				TargetGroupARNs: []string{
					"arn:partition:service:region:account-id:resource-type/resource-id", // link
				},
				HealthCheckType:        adapterhelpers.PtrString("EC2"),
				HealthCheckGracePeriod: adapterhelpers.PtrInt32(15),
				Instances: []types.Instance{
					{
						InstanceId:       adapterhelpers.PtrString("i-0be6c4fe789cb1b78"), // link
						InstanceType:     adapterhelpers.PtrString("t3.large"),
						AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
						LifecycleState:   types.LifecycleStateInService,
						HealthStatus:     adapterhelpers.PtrString("Healthy"),
						LaunchTemplate: &types.LaunchTemplateSpecification{
							LaunchTemplateId:   adapterhelpers.PtrString("lt-0174ff2b8909d0c75"), // Link
							LaunchTemplateName: adapterhelpers.PtrString("eks-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
							Version:            adapterhelpers.PtrString("1"),
						},
						ProtectedFromScaleIn: adapterhelpers.PtrBool(false),
					},
				},
				CreatedTime:        adapterhelpers.PtrTime(time.Now()),
				SuspendedProcesses: []types.SuspendedProcess{},
				VPCZoneIdentifier:  adapterhelpers.PtrString("subnet-0e234bef35fc4a9e1,subnet-09d5f6fa75b0b4569,subnet-0960234bbc4edca03"),
				EnabledMetrics:     []types.EnabledMetric{},
				Tags: []types.TagDescription{
					{
						ResourceId:        adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      adapterhelpers.PtrString("auto-scaling-group"),
						Key:               adapterhelpers.PtrString("eks:cluster-name"),
						Value:             adapterhelpers.PtrString("dogfood"),
						PropagateAtLaunch: adapterhelpers.PtrBool(true),
					},
					{
						ResourceId:        adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      adapterhelpers.PtrString("auto-scaling-group"),
						Key:               adapterhelpers.PtrString("eks:nodegroup-name"),
						Value:             adapterhelpers.PtrString("default-20230117110031319900000013"),
						PropagateAtLaunch: adapterhelpers.PtrBool(true),
					},
					{
						ResourceId:        adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      adapterhelpers.PtrString("auto-scaling-group"),
						Key:               adapterhelpers.PtrString("k8s.io/cluster-autoscaler/dogfood"),
						Value:             adapterhelpers.PtrString("owned"),
						PropagateAtLaunch: adapterhelpers.PtrBool(true),
					},
					{
						ResourceId:        adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      adapterhelpers.PtrString("auto-scaling-group"),
						Key:               adapterhelpers.PtrString("k8s.io/cluster-autoscaler/enabled"),
						Value:             adapterhelpers.PtrString("true"),
						PropagateAtLaunch: adapterhelpers.PtrBool(true),
					},
					{
						ResourceId:        adapterhelpers.PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      adapterhelpers.PtrString("auto-scaling-group"),
						Key:               adapterhelpers.PtrString("kubernetes.io/cluster/dogfood"),
						Value:             adapterhelpers.PtrString("owned"),
						PropagateAtLaunch: adapterhelpers.PtrBool(true),
					},
				},
				TerminationPolicies: []string{
					"AllocationStrategy",
					"OldestLaunchTemplate",
					"OldestInstance",
				},
				NewInstancesProtectedFromScaleIn: adapterhelpers.PtrBool(false),
				ServiceLinkedRoleARN:             adapterhelpers.PtrString("arn:aws:iam::944651592624:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"), // link
				CapacityRebalance:                adapterhelpers.PtrBool(true),
				TrafficSources: []types.TrafficSourceIdentifier{
					{
						Identifier: adapterhelpers.PtrString("arn:partition:service:region:account-id:resource-type/resource-id"), // We will skip this for now since it's related to VPC lattice groups which are still in preview
					},
				},
				Context:                 adapterhelpers.PtrString("foo"),
				DefaultInstanceWarmup:   adapterhelpers.PtrInt32(10),
				DesiredCapacityType:     adapterhelpers.PtrString("foo"),
				LaunchConfigurationName: adapterhelpers.PtrString("launchConfig"), // link
				LaunchTemplate: &types.LaunchTemplateSpecification{
					LaunchTemplateId:   adapterhelpers.PtrString("id"), // link
					LaunchTemplateName: adapterhelpers.PtrString("launchTemplateName"),
				},
				MaxInstanceLifetime: adapterhelpers.PtrInt32(30),
				PlacementGroup:      adapterhelpers.PtrString("placementGroup"), // link (ec2)
				PredictedCapacity:   adapterhelpers.PtrInt32(1),
				Status:              adapterhelpers.PtrString("OK"),
				WarmPoolConfiguration: &types.WarmPoolConfiguration{
					InstanceReusePolicy: &types.InstanceReusePolicy{
						ReuseOnScaleIn: adapterhelpers.PtrBool(true),
					},
					MaxGroupPreparedCapacity: adapterhelpers.PtrInt32(1),
					MinSize:                  adapterhelpers.PtrInt32(1),
					PoolState:                types.WarmPoolStateHibernated,
					Status:                   types.WarmPoolStatusPendingDelete,
				},
				WarmPoolSize: adapterhelpers.PtrInt32(1),
			},
		},
	}

	items, err := autoScalingGroupOutputMapper(context.Background(), nil, "foo", nil, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-launch-template",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "lt-0174ff2b8909d0c75",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type/resource-id",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-0be6c4fe789cb1b78",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::944651592624:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling",
			ExpectedScope:  "944651592624",
		},
		{
			ExpectedType:   "autoscaling-launch-configuration",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "launchConfig",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-launch-template",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-placement-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "placementGroup",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-launch-template",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "lt-0174ff2b8909d0c75",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}
