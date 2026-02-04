package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestAutoScalingGroupOutputMapper(t *testing.T) {
	t.Parallel()

	output := autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []types.AutoScalingGroup{
			{
				AutoScalingGroupName: PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
				AutoScalingGroupARN:  PtrString("arn:aws:autoscaling:eu-west-2:944651592624:autoScalingGroup:1cbb0e22-818f-4d8b-8662-77f73d3713ca:autoScalingGroupName/eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
				MixedInstancesPolicy: &types.MixedInstancesPolicy{
					LaunchTemplate: &types.LaunchTemplate{
						LaunchTemplateSpecification: &types.LaunchTemplateSpecification{
							LaunchTemplateId:   PtrString("lt-0174ff2b8909d0c75"), // link
							LaunchTemplateName: PtrString("eks-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
							Version:            PtrString("1"),
						},
						Overrides: []types.LaunchTemplateOverrides{
							{
								InstanceType: PtrString("t3.large"),
							},
						},
					},
					InstancesDistribution: &types.InstancesDistribution{
						OnDemandAllocationStrategy:          PtrString("prioritized"),
						OnDemandBaseCapacity:                PtrInt32(0),
						OnDemandPercentageAboveBaseCapacity: PtrInt32(100),
						SpotAllocationStrategy:              PtrString("lowest-price"),
						SpotInstancePools:                   PtrInt32(2),
					},
				},
				MinSize:         PtrInt32(1),
				MaxSize:         PtrInt32(3),
				DesiredCapacity: PtrInt32(1),
				DefaultCooldown: PtrInt32(300),
				AvailabilityZones: []string{ // link
					"eu-west-2c",
					"eu-west-2a",
					"eu-west-2b",
				},
				LoadBalancerNames: []string{}, // Ignored, classic load balancer
				TargetGroupARNs: []string{
					"arn:partition:service:region:account-id:resource-type/resource-id", // link
				},
				HealthCheckType:        PtrString("EC2"),
				HealthCheckGracePeriod: PtrInt32(15),
				Instances: []types.Instance{
					{
						InstanceId:       PtrString("i-0be6c4fe789cb1b78"), // link
						InstanceType:     PtrString("t3.large"),
						AvailabilityZone: PtrString("eu-west-2c"),
						LifecycleState:   types.LifecycleStateInService,
						HealthStatus:     PtrString("Healthy"),
						LaunchTemplate: &types.LaunchTemplateSpecification{
							LaunchTemplateId:   PtrString("lt-0174ff2b8909d0c75"), // Link
							LaunchTemplateName: PtrString("eks-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
							Version:            PtrString("1"),
						},
						ProtectedFromScaleIn: PtrBool(false),
					},
				},
				CreatedTime:        PtrTime(time.Now()),
				SuspendedProcesses: []types.SuspendedProcess{},
				VPCZoneIdentifier:  PtrString("subnet-0e234bef35fc4a9e1,subnet-09d5f6fa75b0b4569,subnet-0960234bbc4edca03"),
				EnabledMetrics:     []types.EnabledMetric{},
				Tags: []types.TagDescription{
					{
						ResourceId:        PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      PtrString("auto-scaling-group"),
						Key:               PtrString("eks:cluster-name"),
						Value:             PtrString("dogfood"),
						PropagateAtLaunch: PtrBool(true),
					},
					{
						ResourceId:        PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      PtrString("auto-scaling-group"),
						Key:               PtrString("eks:nodegroup-name"),
						Value:             PtrString("default-20230117110031319900000013"),
						PropagateAtLaunch: PtrBool(true),
					},
					{
						ResourceId:        PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      PtrString("auto-scaling-group"),
						Key:               PtrString("k8s.io/cluster-autoscaler/dogfood"),
						Value:             PtrString("owned"),
						PropagateAtLaunch: PtrBool(true),
					},
					{
						ResourceId:        PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      PtrString("auto-scaling-group"),
						Key:               PtrString("k8s.io/cluster-autoscaler/enabled"),
						Value:             PtrString("true"),
						PropagateAtLaunch: PtrBool(true),
					},
					{
						ResourceId:        PtrString("eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"),
						ResourceType:      PtrString("auto-scaling-group"),
						Key:               PtrString("kubernetes.io/cluster/dogfood"),
						Value:             PtrString("owned"),
						PropagateAtLaunch: PtrBool(true),
					},
				},
				TerminationPolicies: []string{
					"AllocationStrategy",
					"OldestLaunchTemplate",
					"OldestInstance",
				},
				NewInstancesProtectedFromScaleIn: PtrBool(false),
				ServiceLinkedRoleARN:             PtrString("arn:aws:iam::944651592624:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"), // link
				CapacityRebalance:                PtrBool(true),
				TrafficSources: []types.TrafficSourceIdentifier{
					{
						Identifier: PtrString("arn:partition:service:region:account-id:resource-type/resource-id"), // We will skip this for now since it's related to VPC lattice groups which are still in preview
					},
				},
				Context:                 PtrString("foo"),
				DefaultInstanceWarmup:   PtrInt32(10),
				DesiredCapacityType:     PtrString("foo"),
				LaunchConfigurationName: PtrString("launchConfig"), // link
				LaunchTemplate: &types.LaunchTemplateSpecification{
					LaunchTemplateId:   PtrString("id"), // link
					LaunchTemplateName: PtrString("launchTemplateName"),
				},
				MaxInstanceLifetime: PtrInt32(30),
				PlacementGroup:      PtrString("placementGroup"), // link (ec2)
				PredictedCapacity:   PtrInt32(1),
				Status:              PtrString("OK"),
				WarmPoolConfiguration: &types.WarmPoolConfiguration{
					InstanceReusePolicy: &types.InstanceReusePolicy{
						ReuseOnScaleIn: PtrBool(true),
					},
					MaxGroupPreparedCapacity: PtrInt32(1),
					MinSize:                  PtrInt32(1),
					PoolState:                types.WarmPoolStateHibernated,
					Status:                   types.WarmPoolStatusPendingDelete,
				},
				WarmPoolSize: PtrInt32(1),
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
	tests := QueryTests{
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

func TestAutoScalingGroupInputMapperSearch(t *testing.T) {
	t.Parallel()

	adapter := NewAutoScalingGroupAdapter(&autoscaling.Client{}, "123456789012", "us-east-1", sdpcache.NewNoOpCache())

	tests := []struct {
		name          string
		query         string
		expectedNames []string
		expectError   bool
	}{
		{
			name:          "Valid AutoScaling Group ARN",
			query:         "arn:aws:autoscaling:eu-west-2:123456789012:autoScalingGroup:1cbb0e22-818f-4d8b-8662-77f73d3713ca:autoScalingGroupName/eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c",
			expectedNames: []string{"eks-default-20230117110031319900000013-96c2dfb1-a11b-b5e4-6efb-0fea7e22855c"},
			expectError:   false,
		},
		{
			name:          "Valid AutoScaling Group ARN with hyphenated name",
			query:         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:abcd1234-5678-90ab-cdef-1234567890ab:autoScalingGroupName/CodeDeploy_sis_imports_adp_worker_d-MUAZOWH2E",
			expectedNames: []string{"CodeDeploy_sis_imports_adp_worker_d-MUAZOWH2E"},
			expectError:   false,
		},
		{
			name:        "Invalid ARN - not autoscaling service",
			query:       "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			expectError: true,
		},
		{
			name:        "Invalid ARN - malformed",
			query:       "not-an-arn/malformed",
			expectError: true,
		},
		{
			name:        "Invalid ARN - missing autoScalingGroupName",
			query:       "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:abcd1234-5678-90ab-cdef-1234567890ab",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input, err := adapter.InputMapperSearch(context.Background(), &autoscaling.Client{}, "123456789012.us-east-1", tt.query)

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
				t.Errorf("Expected non-nil input for query %s", tt.query)
				return
			}

			if len(input.AutoScalingGroupNames) != len(tt.expectedNames) {
				t.Errorf("Expected %d AutoScalingGroupNames, got %d. Expected: %v, Actual: %v", len(tt.expectedNames), len(input.AutoScalingGroupNames), tt.expectedNames, input.AutoScalingGroupNames)
				return
			}

			for i, expectedName := range tt.expectedNames {
				if input.AutoScalingGroupNames[i] != expectedName {
					t.Errorf("Expected AutoScalingGroupName %s at index %d, got %s", expectedName, i, input.AutoScalingGroupNames[i])
				}
			}
		})
	}
}
