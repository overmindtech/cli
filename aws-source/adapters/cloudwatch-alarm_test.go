package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

type testCloudwatchClient struct{}

func (c testCloudwatchClient) ListTagsForResource(ctx context.Context, params *cloudwatch.ListTagsForResourceInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
	return &cloudwatch.ListTagsForResourceOutput{
		Tags: []types.Tag{
			{
				Key:   new("Name"),
				Value: new("example"),
			},
		},
	}, nil
}

func (c testCloudwatchClient) DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return nil, nil
}

func (c testCloudwatchClient) DescribeAlarmsForMetric(ctx context.Context, params *cloudwatch.DescribeAlarmsForMetricInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsForMetricOutput, error) {
	return nil, nil
}

func TestAlarmOutputMapper(t *testing.T) {
	output := &cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{
			{
				AlarmName:                          new("TargetTracking-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmArn:                           new("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmDescription:                   new("DO NOT EDIT OR DELETE. For TargetTrackingScaling policy arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b."),
				AlarmConfigurationUpdatedTimestamp: new(time.Now()),
				ActionsEnabled:                     new(true),
				OKActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				AlarmActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				InsufficientDataActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				StateValue:            types.StateValueOk,
				StateReason:           new("Threshold Crossed: 2 datapoints [0.0 (09/01/23 14:02:00), 1.0 (09/01/23 14:01:00)] were not greater than the threshold (42.0)."),
				StateReasonData:       new("{\"version\":\"1.0\",\"queryDate\":\"2023-01-09T14:07:25.504+0000\",\"startDate\":\"2023-01-09T14:01:00.000+0000\",\"statistic\":\"Sum\",\"period\":60,\"recentDatapoints\":[1.0,0.0],\"threshold\":42.0,\"evaluatedDatapoints\":[{\"timestamp\":\"2023-01-09T14:02:00.000+0000\",\"sampleCount\":1.0,\"value\":0.0}]}"),
				StateUpdatedTimestamp: new(time.Now()),
				MetricName:            new("ConsumedWriteCapacityUnits"),
				Namespace:             new("AWS/DynamoDB"),
				Statistic:             types.StatisticSum,
				Dimensions: []types.Dimension{
					{
						Name:  new("TableName"),
						Value: new("dylan-tfstate"),
					},
				},
				Period:                     new(int32(60)),
				EvaluationPeriods:          new(int32(2)),
				Threshold:                  new(42.0),
				ComparisonOperator:         types.ComparisonOperatorGreaterThanThreshold,
				StateTransitionedTimestamp: new(time.Now()),
			},
		},
		CompositeAlarms: []types.CompositeAlarm{
			{
				AlarmName:                          new("TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmArn:                           new("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmDescription:                   new("DO NOT EDIT OR DELETE. For TargetTrackingScaling policy arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b."),
				AlarmConfigurationUpdatedTimestamp: new(time.Now()),
				ActionsEnabled:                     new(true),
				OKActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				AlarmActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				InsufficientDataActions: []string{
					"arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b",
				},
				StateValue:                 types.StateValueOk,
				StateReason:                new("Threshold Crossed: 2 datapoints [0.0 (09/01/23 14:02:00), 1.0 (09/01/23 14:01:00)] were not greater than the threshold (42.0)."),
				StateReasonData:            new("{\"version\":\"1.0\",\"queryDate\":\"2023-01-09T14:07:25.504+0000\",\"startDate\":\"2023-01-09T14:01:00.000+0000\",\"statistic\":\"Sum\",\"period\":60,\"recentDatapoints\":[1.0,0.0],\"threshold\":42.0,\"evaluatedDatapoints\":[{\"timestamp\":\"2023-01-09T14:02:00.000+0000\",\"sampleCount\":1.0,\"value\":0.0}]}"),
				StateUpdatedTimestamp:      new(time.Now()),
				StateTransitionedTimestamp: new(time.Now()),
				ActionsSuppressedBy:        types.ActionsSuppressedByAlarm,
				ActionsSuppressedReason:    new("Alarm is in INSUFFICIENT_DATA state"),
				// link
				ActionsSuppressor:                new("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				ActionsSuppressorExtensionPeriod: new(int32(0)),
				ActionsSuppressorWaitPeriod:      new(int32(0)),
				AlarmRule:                        new("ALARM TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
			},
		},
	}

	scope := "123456789012.eu-west-2"
	items, err := alarmOutputMapper(context.Background(), testCloudwatchClient{}, scope, &cloudwatch.DescribeAlarmsInput{}, output)

	if err != nil {
		t.Error(err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	item := items[1]

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	if item.GetTags()["Name"] != "example" {
		t.Errorf("Expected tag Name to be example, got %s", item.GetTags()["Name"])
	}

	tests := QueryTests{
		{
			ExpectedType:   "cloudwatch-alarm",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d",
			ExpectedScope:  "052392120703.eu-west-2",
		},
	}

	tests.Execute(t, item)

	item = items[0]

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests = QueryTests{
		{
			ExpectedType:   "dynamodb-table",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dylan-tfstate",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

// testCloudwatchClientWithTagError returns an error when fetching tags
// to simulate scenarios where tag access is denied but alarm data is available
type testCloudwatchClientWithTagError struct{}

func (c testCloudwatchClientWithTagError) ListTagsForResource(ctx context.Context, params *cloudwatch.ListTagsForResourceInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
	return nil, fmt.Errorf("access denied: cannot list tags for resource")
}

func (c testCloudwatchClientWithTagError) DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return nil, nil
}

func (c testCloudwatchClientWithTagError) DescribeAlarmsForMetric(ctx context.Context, params *cloudwatch.DescribeAlarmsForMetricInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsForMetricOutput, error) {
	return nil, nil
}

// TestAlarmOutputMapperWithTagError tests that items are still returned when
// tag fetching fails. This is a regression test for a bug where a leftover
// error check caused the mapper to return nil items when ListTagsForResource
// failed, even though the alarm data was successfully retrieved.
func TestAlarmOutputMapperWithTagError(t *testing.T) {
	output := &cloudwatch.DescribeAlarmsOutput{
		MetricAlarms: []types.MetricAlarm{
			{
				AlarmName:        new("api-51c748b4-cpu-credits-low"),
				AlarmArn:         new("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:api-51c748b4-cpu-credits-low"),
				AlarmDescription: new("CPU credits low alarm"),
				StateValue:       types.StateValueOk,
				MetricName:       new("CPUCreditBalance"),
				Namespace:        new("AWS/EC2"),
			},
		},
	}

	scope := "123456789012.eu-west-2"
	// Use the client that returns an error when fetching tags
	items, err := alarmOutputMapper(context.Background(), testCloudwatchClientWithTagError{}, scope, &cloudwatch.DescribeAlarmsInput{}, output)

	if err != nil {
		t.Errorf("Expected no error when tag fetching fails, but got: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item to be returned even when tag fetching fails, got %d", len(items))
	}

	item := items[0]
	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	// Verify the alarm name is correct
	alarmName, err := item.GetAttributes().Get("AlarmName")
	if err != nil {
		t.Errorf("Failed to get AlarmName: %v", err)
	}
	if alarmName != "api-51c748b4-cpu-credits-low" {
		t.Errorf("Expected AlarmName to be 'api-51c748b4-cpu-credits-low', got %v", alarmName)
	}
}

func TestNewCloudwatchAlarmAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := cloudwatch.NewFromConfig(config)

	adapter := NewCloudwatchAlarmAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
