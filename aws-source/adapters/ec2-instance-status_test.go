package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestInstanceStatusInputMapperGet(t *testing.T) {
	input, err := instanceStatusInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.InstanceIds) != 1 {
		t.Fatalf("expected 1 instanceStatus ID, got %v", len(input.InstanceIds))
	}

	if input.InstanceIds[0] != "bar" {
		t.Errorf("expected instanceStatus ID to be bar, got %v", input.InstanceIds[0])
	}
}

func TestInstanceStatusInputMapperList(t *testing.T) {
	input, err := instanceStatusInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.InstanceIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestInstanceStatusOutputMapper(t *testing.T) {
	output := &ec2.DescribeInstanceStatusOutput{
		InstanceStatuses: []types.InstanceStatus{
			{
				AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),          // link
				InstanceId:       adapterhelpers.PtrString("i-022bdccde30270570"), // link
				InstanceState: &types.InstanceState{
					Code: adapterhelpers.PtrInt32(16),
					Name: types.InstanceStateNameRunning,
				},
				InstanceStatus: &types.InstanceStatusSummary{
					Details: []types.InstanceStatusDetails{
						{
							Name:   types.StatusNameReachability,
							Status: types.StatusTypePassed,
						},
					},
					Status: types.SummaryStatusOk,
				},
				SystemStatus: &types.InstanceStatusSummary{
					Details: []types.InstanceStatusDetails{
						{
							Name:   types.StatusNameReachability,
							Status: types.StatusTypePassed,
						},
					},
					Status: types.SummaryStatusImpaired,
				},
			},
		},
	}

	items, err := instanceStatusOutputMapper(context.Background(), nil, "foo", nil, output)

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

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-022bdccde30270570",
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2InstanceStatusAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2InstanceStatusAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
