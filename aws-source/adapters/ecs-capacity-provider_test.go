package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func (t *ecsTestClient) DescribeCapacityProviders(ctx context.Context, params *ecs.DescribeCapacityProvidersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeCapacityProvidersOutput, error) {
	pages := map[string]*ecs.DescribeCapacityProvidersOutput{
		"": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: new("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/FARGATE"),
					Name:                new("FARGATE"),
					Status:              types.CapacityProviderStatusActive,
				},
			},
			NextToken: new("one"),
		},
		"one": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: new("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/FARGATE_SPOT"),
					Name:                new("FARGATE_SPOT"),
					Status:              types.CapacityProviderStatusActive,
				},
			},
			NextToken: new("two"),
		},
		"two": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: new("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/test"),
					Name:                new("test"),
					Status:              types.CapacityProviderStatusActive,
					AutoScalingGroupProvider: &types.AutoScalingGroupProvider{
						AutoScalingGroupArn: new("arn:aws:autoscaling:eu-west-2:052392120703:autoScalingGroup:9df90815-98c1-4136-a12a-90abef1c4e4e:autoScalingGroupName/ecs-test"),
						ManagedScaling: &types.ManagedScaling{
							Status:                 types.ManagedScalingStatusEnabled,
							TargetCapacity:         new(int32(80)),
							MinimumScalingStepSize: new(int32(1)),
							MaximumScalingStepSize: new(int32(10000)),
							InstanceWarmupPeriod:   new(int32(300)),
						},
						ManagedTerminationProtection: types.ManagedTerminationProtectionDisabled,
					},
					UpdateStatus:       types.CapacityProviderUpdateStatusDeleteComplete,
					UpdateStatusReason: new("reason"),
				},
			},
		},
	}

	var page string

	if params.NextToken != nil {
		page = *params.NextToken
	}

	return pages[page], nil
}

func TestCapacityProviderOutputMapper(t *testing.T) {
	items, err := capacityProviderOutputMapper(
		context.Background(),
		&ecsTestClient{},
		"foo",
		nil,
		&ecs.DescribeCapacityProvidersOutput{
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: new("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/test"),
					Name:                new("test"),
					Status:              types.CapacityProviderStatusActive,
					AutoScalingGroupProvider: &types.AutoScalingGroupProvider{
						AutoScalingGroupArn: new("arn:aws:autoscaling:eu-west-2:052392120703:autoScalingGroup:9df90815-98c1-4136-a12a-90abef1c4e4e:autoScalingGroupName/ecs-test"),
						ManagedScaling: &types.ManagedScaling{
							Status:                 types.ManagedScalingStatusEnabled,
							TargetCapacity:         new(int32(80)),
							MinimumScalingStepSize: new(int32(1)),
							MaximumScalingStepSize: new(int32(10000)),
							InstanceWarmupPeriod:   new(int32(300)),
						},
						ManagedTerminationProtection: types.ManagedTerminationProtectionDisabled,
					},
					UpdateStatus:       types.CapacityProviderUpdateStatusDeleteComplete,
					UpdateStatusReason: new("reason"),
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "autoscaling-auto-scaling-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:autoscaling:eu-west-2:052392120703:autoScalingGroup:9df90815-98c1-4136-a12a-90abef1c4e4e:autoScalingGroupName/ecs-test",
			ExpectedScope:  "052392120703.eu-west-2",
		},
	}

	tests.Execute(t, item)
}

func TestCapacityProviderAdapter(t *testing.T) {
	adapter := NewECSCapacityProviderAdapter(&ecsTestClient{}, "", "", sdpcache.NewNoOpCache())

	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(context.Background(), "*", false, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		t.Error(errs)
	}

	items := stream.GetItems()
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %v", len(items))
	}
}

func TestNewECSCapacityProviderAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := ecs.NewFromConfig(config)

	adapter := NewECSCapacityProviderAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
