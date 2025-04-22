package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *ecsTestClient) DescribeCapacityProviders(ctx context.Context, params *ecs.DescribeCapacityProvidersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeCapacityProvidersOutput, error) {
	pages := map[string]*ecs.DescribeCapacityProvidersOutput{
		"": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: adapterhelpers.PtrString("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/FARGATE"),
					Name:                adapterhelpers.PtrString("FARGATE"),
					Status:              types.CapacityProviderStatusActive,
				},
			},
			NextToken: adapterhelpers.PtrString("one"),
		},
		"one": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: adapterhelpers.PtrString("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/FARGATE_SPOT"),
					Name:                adapterhelpers.PtrString("FARGATE_SPOT"),
					Status:              types.CapacityProviderStatusActive,
				},
			},
			NextToken: adapterhelpers.PtrString("two"),
		},
		"two": {
			CapacityProviders: []types.CapacityProvider{
				{
					CapacityProviderArn: adapterhelpers.PtrString("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/test"),
					Name:                adapterhelpers.PtrString("test"),
					Status:              types.CapacityProviderStatusActive,
					AutoScalingGroupProvider: &types.AutoScalingGroupProvider{
						AutoScalingGroupArn: adapterhelpers.PtrString("arn:aws:autoscaling:eu-west-2:052392120703:autoScalingGroup:9df90815-98c1-4136-a12a-90abef1c4e4e:autoScalingGroupName/ecs-test"),
						ManagedScaling: &types.ManagedScaling{
							Status:                 types.ManagedScalingStatusEnabled,
							TargetCapacity:         adapterhelpers.PtrInt32(80),
							MinimumScalingStepSize: adapterhelpers.PtrInt32(1),
							MaximumScalingStepSize: adapterhelpers.PtrInt32(10000),
							InstanceWarmupPeriod:   adapterhelpers.PtrInt32(300),
						},
						ManagedTerminationProtection: types.ManagedTerminationProtectionDisabled,
					},
					UpdateStatus:       types.CapacityProviderUpdateStatusDeleteComplete,
					UpdateStatusReason: adapterhelpers.PtrString("reason"),
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
					CapacityProviderArn: adapterhelpers.PtrString("arn:aws:ecs:eu-west-2:052392120703:capacity-provider/test"),
					Name:                adapterhelpers.PtrString("test"),
					Status:              types.CapacityProviderStatusActive,
					AutoScalingGroupProvider: &types.AutoScalingGroupProvider{
						AutoScalingGroupArn: adapterhelpers.PtrString("arn:aws:autoscaling:eu-west-2:052392120703:autoScalingGroup:9df90815-98c1-4136-a12a-90abef1c4e4e:autoScalingGroupName/ecs-test"),
						ManagedScaling: &types.ManagedScaling{
							Status:                 types.ManagedScalingStatusEnabled,
							TargetCapacity:         adapterhelpers.PtrInt32(80),
							MinimumScalingStepSize: adapterhelpers.PtrInt32(1),
							MaximumScalingStepSize: adapterhelpers.PtrInt32(10000),
							InstanceWarmupPeriod:   adapterhelpers.PtrInt32(300),
						},
						ManagedTerminationProtection: types.ManagedTerminationProtectionDisabled,
					},
					UpdateStatus:       types.CapacityProviderUpdateStatusDeleteComplete,
					UpdateStatusReason: adapterhelpers.PtrString("reason"),
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

	tests := adapterhelpers.QueryTests{
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
	adapter := NewECSCapacityProviderAdapter(&ecsTestClient{}, "", "")

	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(context.Background(), "", false, stream)

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
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := ecs.NewFromConfig(config)

	adapter := NewECSCapacityProviderAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
