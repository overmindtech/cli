package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type testCloudwatchClient struct{}

func (c testCloudwatchClient) ListTagsForResource(ctx context.Context, params *cloudwatch.ListTagsForResourceInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListTagsForResourceOutput, error) {
	return &cloudwatch.ListTagsForResourceOutput{
		Tags: []types.Tag{
			{
				Key:   adapterhelpers.PtrString("Name"),
				Value: adapterhelpers.PtrString("example"),
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
				AlarmName:                          adapterhelpers.PtrString("TargetTracking-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmArn:                           adapterhelpers.PtrString("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmDescription:                   adapterhelpers.PtrString("DO NOT EDIT OR DELETE. For TargetTrackingScaling policy arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b."),
				AlarmConfigurationUpdatedTimestamp: adapterhelpers.PtrTime(time.Now()),
				ActionsEnabled:                     adapterhelpers.PtrBool(true),
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
				StateReason:           adapterhelpers.PtrString("Threshold Crossed: 2 datapoints [0.0 (09/01/23 14:02:00), 1.0 (09/01/23 14:01:00)] were not greater than the threshold (42.0)."),
				StateReasonData:       adapterhelpers.PtrString("{\"version\":\"1.0\",\"queryDate\":\"2023-01-09T14:07:25.504+0000\",\"startDate\":\"2023-01-09T14:01:00.000+0000\",\"statistic\":\"Sum\",\"period\":60,\"recentDatapoints\":[1.0,0.0],\"threshold\":42.0,\"evaluatedDatapoints\":[{\"timestamp\":\"2023-01-09T14:02:00.000+0000\",\"sampleCount\":1.0,\"value\":0.0}]}"),
				StateUpdatedTimestamp: adapterhelpers.PtrTime(time.Now()),
				MetricName:            adapterhelpers.PtrString("ConsumedWriteCapacityUnits"),
				Namespace:             adapterhelpers.PtrString("AWS/DynamoDB"),
				Statistic:             types.StatisticSum,
				Dimensions: []types.Dimension{
					{
						Name:  adapterhelpers.PtrString("TableName"),
						Value: adapterhelpers.PtrString("dylan-tfstate"),
					},
				},
				Period:                     adapterhelpers.PtrInt32(60),
				EvaluationPeriods:          adapterhelpers.PtrInt32(2),
				Threshold:                  adapterhelpers.PtrFloat64(42.0),
				ComparisonOperator:         types.ComparisonOperatorGreaterThanThreshold,
				StateTransitionedTimestamp: adapterhelpers.PtrTime(time.Now()),
			},
		},
		CompositeAlarms: []types.CompositeAlarm{
			{
				AlarmName:                          adapterhelpers.PtrString("TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmArn:                           adapterhelpers.PtrString("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				AlarmDescription:                   adapterhelpers.PtrString("DO NOT EDIT OR DELETE. For TargetTrackingScaling policy arn:aws:autoscaling:eu-west-2:052392120703:scalingPolicy:32f3f053-dc75-46fa-9cd4-8e8c34c47b37:resource/dynamodb/table/dylan-tfstate:policyName/$dylan-tfstate-scaling-policy:createdBy/e5bd51d8-94a8-461e-a989-08f4d10b326b."),
				AlarmConfigurationUpdatedTimestamp: adapterhelpers.PtrTime(time.Now()),
				ActionsEnabled:                     adapterhelpers.PtrBool(true),
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
				StateReason:                adapterhelpers.PtrString("Threshold Crossed: 2 datapoints [0.0 (09/01/23 14:02:00), 1.0 (09/01/23 14:01:00)] were not greater than the threshold (42.0)."),
				StateReasonData:            adapterhelpers.PtrString("{\"version\":\"1.0\",\"queryDate\":\"2023-01-09T14:07:25.504+0000\",\"startDate\":\"2023-01-09T14:01:00.000+0000\",\"statistic\":\"Sum\",\"period\":60,\"recentDatapoints\":[1.0,0.0],\"threshold\":42.0,\"evaluatedDatapoints\":[{\"timestamp\":\"2023-01-09T14:02:00.000+0000\",\"sampleCount\":1.0,\"value\":0.0}]}"),
				StateUpdatedTimestamp:      adapterhelpers.PtrTime(time.Now()),
				StateTransitionedTimestamp: adapterhelpers.PtrTime(time.Now()),
				ActionsSuppressedBy:        types.ActionsSuppressedByAlarm,
				ActionsSuppressedReason:    adapterhelpers.PtrString("Alarm is in INSUFFICIENT_DATA state"),
				// link
				ActionsSuppressor:                adapterhelpers.PtrString("arn:aws:cloudwatch:eu-west-2:052392120703:alarm:TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
				ActionsSuppressorExtensionPeriod: adapterhelpers.PtrInt32(0),
				ActionsSuppressorWaitPeriod:      adapterhelpers.PtrInt32(0),
				AlarmRule:                        adapterhelpers.PtrString("ALARM TargetTracking2-table/dylan-tfstate-AlarmHigh-14069c4a-6dcc-48a2-bfe6-b5547c90c43d"),
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

	tests := adapterhelpers.QueryTests{
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

	tests = adapterhelpers.QueryTests{
		{
			ExpectedType:   "dynamodb-table",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dylan-tfstate",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestNewCloudwatchAlarmAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := cloudwatch.NewFromConfig(config)

	adapter := NewCloudwatchAlarmAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
