package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestScalingPolicyOutputMapper(t *testing.T) {
	t.Parallel()

	output := autoscaling.DescribePoliciesOutput{
		ScalingPolicies: []types.ScalingPolicy{
			{
				PolicyName:              adapterhelpers.PtrString("scale-up-policy"),
				PolicyARN:               adapterhelpers.PtrString("arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg:policyName/scale-up-policy"),
				AutoScalingGroupName:    adapterhelpers.PtrString("my-asg"),
				PolicyType:              adapterhelpers.PtrString("TargetTrackingScaling"),
				AdjustmentType:          adapterhelpers.PtrString("ChangeInCapacity"),
				MinAdjustmentMagnitude:  adapterhelpers.PtrInt32(1),
				ScalingAdjustment:       adapterhelpers.PtrInt32(1),
				Cooldown:                adapterhelpers.PtrInt32(300),
				MetricAggregationType:   adapterhelpers.PtrString("Average"),
				EstimatedInstanceWarmup: adapterhelpers.PtrInt32(300),
				Enabled:                 adapterhelpers.PtrBool(true),
				TargetTrackingConfiguration: &types.TargetTrackingConfiguration{
					PredefinedMetricSpecification: &types.PredefinedMetricSpecification{
						PredefinedMetricType: types.MetricTypeALBRequestCountPerTarget,
						ResourceLabel:        adapterhelpers.PtrString("app/my-alb/778d41231b141a0f/targetgroup/my-alb-target-group/943f017f100becff"),
					},
					TargetValue: adapterhelpers.PtrFloat64(50.0),
				},
				Alarms: []types.Alarm{
					{
						AlarmName: adapterhelpers.PtrString("my-alarm-high"),
						AlarmARN:  adapterhelpers.PtrString("arn:aws:cloudwatch:us-east-1:123456789012:alarm:my-alarm-high"),
					},
					{
						AlarmName: adapterhelpers.PtrString("my-alarm-low"),
						AlarmARN:  adapterhelpers.PtrString("arn:aws:cloudwatch:us-east-1:123456789012:alarm:my-alarm-low"),
					},
				},
			},
			{
				PolicyName:              adapterhelpers.PtrString("step-scaling-policy"),
				PolicyARN:               adapterhelpers.PtrString("arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:87654321-4321-4321-4321-210987654321:autoScalingGroupName/my-asg:policyName/step-scaling-policy"),
				AutoScalingGroupName:    adapterhelpers.PtrString("my-asg"),
				PolicyType:              adapterhelpers.PtrString("StepScaling"),
				AdjustmentType:          adapterhelpers.PtrString("PercentChangeInCapacity"),
				MinAdjustmentMagnitude:  adapterhelpers.PtrInt32(2),
				MetricAggregationType:   adapterhelpers.PtrString("Average"),
				EstimatedInstanceWarmup: adapterhelpers.PtrInt32(60),
				Enabled:                 adapterhelpers.PtrBool(true),
				StepAdjustments: []types.StepAdjustment{
					{
						MetricIntervalLowerBound: adapterhelpers.PtrFloat64(0.0),
						MetricIntervalUpperBound: adapterhelpers.PtrFloat64(10.0),
						ScalingAdjustment:        adapterhelpers.PtrInt32(10),
					},
					{
						MetricIntervalLowerBound: adapterhelpers.PtrFloat64(10.0),
						ScalingAdjustment:        adapterhelpers.PtrInt32(20),
					},
				},
				Alarms: []types.Alarm{
					{
						AlarmName: adapterhelpers.PtrString("step-alarm"),
						AlarmARN:  adapterhelpers.PtrString("arn:aws:cloudwatch:us-east-1:123456789012:alarm:step-alarm"),
					},
				},
			},
			{
				PolicyName:           adapterhelpers.PtrString("simple-scaling-policy"),
				PolicyARN:            adapterhelpers.PtrString("arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:11111111-2222-3333-4444-555555555555:autoScalingGroupName/another-asg:policyName/simple-scaling-policy"),
				AutoScalingGroupName: adapterhelpers.PtrString("another-asg"),
				PolicyType:           adapterhelpers.PtrString("SimpleScaling"),
				AdjustmentType:       adapterhelpers.PtrString("ExactCapacity"),
				ScalingAdjustment:    adapterhelpers.PtrInt32(5),
				Cooldown:             adapterhelpers.PtrInt32(600),
				Enabled:              adapterhelpers.PtrBool(false),
			},
			{
				PolicyName:           adapterhelpers.PtrString("predictive-scaling-policy"),
				PolicyARN:            adapterhelpers.PtrString("arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:99999999-8888-7777-6666-555555555555:autoScalingGroupName/predictive-asg:policyName/predictive-scaling-policy"),
				AutoScalingGroupName: adapterhelpers.PtrString("predictive-asg"),
				PolicyType:           adapterhelpers.PtrString("PredictiveScaling"),
				Enabled:              adapterhelpers.PtrBool(true),
				PredictiveScalingConfiguration: &types.PredictiveScalingConfiguration{
					MetricSpecifications: []types.PredictiveScalingMetricSpecification{
						{
							TargetValue: adapterhelpers.PtrFloat64(40.0),
							PredefinedMetricPairSpecification: &types.PredictiveScalingPredefinedMetricPair{
								PredefinedMetricType: types.PredefinedMetricPairTypeALBRequestCount,
								ResourceLabel:        adapterhelpers.PtrString("app/predictive-alb/abc123def456/targetgroup/predictive-tg/789xyz"),
							},
						},
					},
					Mode: types.PredictiveScalingModeForecastAndScale,
				},
			},
		},
	}

	items, err := scalingPolicyOutputMapper(context.Background(), nil, "test-scope", nil, &output)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(items) != 4 {
		t.Errorf("Expected 4 items, got %v", len(items))
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Errorf("Item validation failed: %v", err)
		}
	}

	// Test the first policy (TargetTrackingScaling with multiple alarms)
	item := items[0]
	if item.GetType() != "autoscaling-auto-scaling-policy" {
		t.Errorf("Expected type 'autoscaling-auto-scaling-policy', got '%v'", item.GetType())
	}

	if item.GetUniqueAttribute() != "UniqueName" {
		t.Errorf("Expected unique attribute 'UniqueName', got '%v'", item.GetUniqueAttribute())
	}

	// Verify the UniqueName attribute is set correctly (asgName/policyName format)
	uniqueName, err := item.GetAttributes().Get("UniqueName")
	if err != nil {
		t.Errorf("Expected UniqueName attribute to be set: %v", err)
	}
	if uniqueName != "my-asg/scale-up-policy" {
		t.Errorf("Expected UniqueName 'my-asg/scale-up-policy', got '%v'", uniqueName)
	}

	// Check linked items
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-asg",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "cloudwatch-alarm",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-alarm-high",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "cloudwatch-alarm",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-alarm-low",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "elbv2-load-balancer",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-alb",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-alb-target-group",
			ExpectedScope:  "test-scope",
		},
	}

	tests.Execute(t, item)

	// Test the second policy (StepScaling)
	item2 := items[1]
	tests2 := adapterhelpers.QueryTests{
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "my-asg",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "cloudwatch-alarm",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "step-alarm",
			ExpectedScope:  "test-scope",
		},
	}

	tests2.Execute(t, item2)

	// Test the third policy (SimpleScaling with no alarms)
	item3 := items[2]
	tests3 := adapterhelpers.QueryTests{
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "another-asg",
			ExpectedScope:  "test-scope",
		},
	}

	tests3.Execute(t, item3)

	// Verify the third policy has no alarm links
	alarmLinkCount := 0
	for _, link := range item3.GetLinkedItemQueries() {
		if link.GetQuery().GetType() == "cloudwatch-alarm" {
			alarmLinkCount++
		}
	}
	if alarmLinkCount != 0 {
		t.Errorf("Expected 0 alarm links for simple-scaling-policy, got %v", alarmLinkCount)
	}

	// Test the fourth policy (PredictiveScaling with ALB ResourceLabel)
	item4 := items[3]
	tests4 := adapterhelpers.QueryTests{
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "predictive-asg",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "elbv2-load-balancer",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "predictive-alb",
			ExpectedScope:  "test-scope",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "predictive-tg",
			ExpectedScope:  "test-scope",
		},
	}

	tests4.Execute(t, item4)
}

func TestParseResourceLabelLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		resourceLabel  string
		expectedLBName string
		expectedTGName string
		expectedCount  int
	}{
		{
			name:           "Valid ALB resource label",
			resourceLabel:  "app/my-alb/778d41231b141a0f/targetgroup/my-target-group/943f017f100becff",
			expectedLBName: "my-alb",
			expectedTGName: "my-target-group",
			expectedCount:  2,
		},
		{
			name:           "Valid ALB resource label with hyphens",
			resourceLabel:  "app/my-load-balancer-name/abc123/targetgroup/my-tg-name/def456",
			expectedLBName: "my-load-balancer-name",
			expectedTGName: "my-tg-name",
			expectedCount:  2,
		},
		{
			name:           "Valid NLB resource label",
			resourceLabel:  "net/my-nlb/778d41231b141a0f/targetgroup/my-target-group/943f017f100becff",
			expectedLBName: "my-nlb",
			expectedTGName: "my-target-group",
			expectedCount:  2,
		},
		{
			name:           "Valid GLB resource label",
			resourceLabel:  "gwy/my-glb/778d41231b141a0f/targetgroup/my-target-group/943f017f100becff",
			expectedLBName: "my-glb",
			expectedTGName: "my-target-group",
			expectedCount:  2,
		},
		{
			name:           "Too few sections",
			resourceLabel:  "app/my-alb/targetgroup",
			expectedCount:  0,
		},
		{
			name:           "Empty string",
			resourceLabel:  "",
			expectedCount:  0,
		},
		{
			name:           "Unknown prefix",
			resourceLabel:  "unknown/my-lb/778d41231b141a0f/targetgroup/my-target-group/943f017f100becff",
			expectedLBName: "",
			expectedTGName: "my-target-group",
			expectedCount:  1, // Only target group, no LB for unknown prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			links := parseResourceLabelLinks(tt.resourceLabel, "test-scope")

			if len(links) != tt.expectedCount {
				t.Errorf("Expected %d links, got %d", tt.expectedCount, len(links))
				return
			}

			if tt.expectedCount == 0 {
				return
			}

			// Check for load balancer link
			if tt.expectedLBName != "" {
				foundLB := false
				for _, link := range links {
					if link.GetQuery().GetType() == "elbv2-load-balancer" {
						foundLB = true
						if link.GetQuery().GetQuery() != tt.expectedLBName {
							t.Errorf("Expected LB name %s, got %s", tt.expectedLBName, link.GetQuery().GetQuery())
						}
						if link.GetQuery().GetScope() != "test-scope" {
							t.Errorf("Expected scope test-scope, got %s", link.GetQuery().GetScope())
						}
					}
				}
				if !foundLB {
					t.Error("Expected load balancer link not found")
				}
			}

			// Check for target group link
			if tt.expectedTGName != "" {
				foundTG := false
				for _, link := range links {
					if link.GetQuery().GetType() == "elbv2-target-group" {
						foundTG = true
						if link.GetQuery().GetQuery() != tt.expectedTGName {
							t.Errorf("Expected TG name %s, got %s", tt.expectedTGName, link.GetQuery().GetQuery())
						}
						if link.GetQuery().GetScope() != "test-scope" {
							t.Errorf("Expected scope test-scope, got %s", link.GetQuery().GetScope())
						}
					}
				}
				if !foundTG {
					t.Error("Expected target group link not found")
				}
			}
		})
	}
}

func TestScalingPolicyInputMapperSearch(t *testing.T) {
	t.Parallel()

	adapter := NewAutoScalingPolicyAdapter(&autoscaling.Client{}, "123456789012", "us-east-1", nil)

	tests := []struct {
		name               string
		query              string
		expectedASGName    string
		expectedPolicyName string
		expectError        bool
	}{
		{
			name:               "Valid Scaling Policy ARN",
			query:              "arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg:policyName/scale-up-policy",
			expectedASGName:    "my-asg",
			expectedPolicyName: "scale-up-policy",
			expectError:        false,
		},
		{
			name:               "Valid Scaling Policy ARN with hyphenated names",
			query:              "arn:aws:autoscaling:eu-west-2:987654321098:scalingPolicy:abcd1234-5678-90ab-cdef-1234567890ab:autoScalingGroupName/my-test-asg-name:policyName/my-test-policy-name",
			expectedASGName:    "my-test-asg-name",
			expectedPolicyName: "my-test-policy-name",
			expectError:        false,
		},
		{
			name:               "Valid Scaling Policy ARN with underscores",
			query:              "arn:aws:autoscaling:ap-southeast-1:111222333444:scalingPolicy:11111111-2222-3333-4444-555555555555:autoScalingGroupName/my_asg_name:policyName/my_policy_name",
			expectedASGName:    "my_asg_name",
			expectedPolicyName: "my_policy_name",
			expectError:        false,
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
			name:        "Invalid ARN - not a scaling policy",
			query:       "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg",
			expectError: true,
		},
		{
			name:        "Invalid ARN - missing autoScalingGroupName",
			query:       "arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:12345678-1234-1234-1234-123456789012:policyName/scale-up-policy",
			expectError: true,
		},
		{
			name:        "Invalid ARN - missing policyName",
			query:       "arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:12345678-1234-1234-1234-123456789012:autoScalingGroupName/my-asg",
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

			if input.AutoScalingGroupName == nil {
				t.Errorf("Expected non-nil AutoScalingGroupName for query %s", tt.query)
				return
			}

			if *input.AutoScalingGroupName != tt.expectedASGName {
				t.Errorf("Expected AutoScalingGroupName %s, got %s", tt.expectedASGName, *input.AutoScalingGroupName)
			}

			if len(input.PolicyNames) != 1 {
				t.Errorf("Expected 1 PolicyName, got %d. PolicyNames: %v", len(input.PolicyNames), input.PolicyNames)
				return
			}

			if input.PolicyNames[0] != tt.expectedPolicyName {
				t.Errorf("Expected PolicyName %s, got %s", tt.expectedPolicyName, input.PolicyNames[0])
			}
		})
	}
}

func TestScalingPolicyInputMapperGet(t *testing.T) {
	t.Parallel()

	adapter := NewAutoScalingPolicyAdapter(&autoscaling.Client{}, "123456789012", "us-east-1", nil)

	tests := []struct {
		name               string
		query              string
		expectedASGName    string
		expectedPolicyName string
		expectError        bool
	}{
		{
			name:               "Valid composite key",
			query:              "my-asg/scale-up-policy",
			expectedASGName:    "my-asg",
			expectedPolicyName: "scale-up-policy",
			expectError:        false,
		},
		{
			name:               "Valid composite key with hyphenated names",
			query:              "my-test-asg-name/my-test-policy-name",
			expectedASGName:    "my-test-asg-name",
			expectedPolicyName: "my-test-policy-name",
			expectError:        false,
		},
		{
			name:               "Valid composite key with underscores",
			query:              "my_asg_name/my_policy_name",
			expectedASGName:    "my_asg_name",
			expectedPolicyName: "my_policy_name",
			expectError:        false,
		},
		{
			name:               "Valid composite key with slashes in policy name",
			query:              "my-asg/path/to/policy",
			expectedASGName:    "my-asg",
			expectedPolicyName: "path/to/policy",
			expectError:        false,
		},
		{
			name:        "Invalid - missing policy name",
			query:       "my-asg/",
			expectError: true,
		},
		{
			name:        "Invalid - missing ASG name",
			query:       "/scale-up-policy",
			expectError: true,
		},
		{
			name:        "Invalid - no slash separator",
			query:       "just-a-policy-name",
			expectError: true,
		},
		{
			name:        "Invalid - empty string",
			query:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input, err := adapter.InputMapperGet("123456789012.us-east-1", tt.query)

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

			if input.AutoScalingGroupName == nil {
				t.Errorf("Expected non-nil AutoScalingGroupName for query %s", tt.query)
				return
			}

			if *input.AutoScalingGroupName != tt.expectedASGName {
				t.Errorf("Expected AutoScalingGroupName %s, got %s", tt.expectedASGName, *input.AutoScalingGroupName)
			}

			if len(input.PolicyNames) != 1 {
				t.Errorf("Expected 1 PolicyName, got %d. PolicyNames: %v", len(input.PolicyNames), input.PolicyNames)
				return
			}

			if input.PolicyNames[0] != tt.expectedPolicyName {
				t.Errorf("Expected PolicyName %s, got %s", tt.expectedPolicyName, input.PolicyNames[0])
			}
		})
	}
}
