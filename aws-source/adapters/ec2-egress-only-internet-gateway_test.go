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

func TestEgressOnlyInternetGatewayInputMapperGet(t *testing.T) {
	input, err := egressOnlyInternetGatewayInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.EgressOnlyInternetGatewayIds) != 1 {
		t.Fatalf("expected 1 EgressOnlyInternetGateway ID, got %v", len(input.EgressOnlyInternetGatewayIds))
	}

	if input.EgressOnlyInternetGatewayIds[0] != "bar" {
		t.Errorf("expected EgressOnlyInternetGateway ID to be bar, got %v", input.EgressOnlyInternetGatewayIds[0])
	}
}

func TestEgressOnlyInternetGatewayInputMapperList(t *testing.T) {
	input, err := egressOnlyInternetGatewayInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.EgressOnlyInternetGatewayIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestEgressOnlyInternetGatewayOutputMapper(t *testing.T) {
	output := &ec2.DescribeEgressOnlyInternetGatewaysOutput{
		EgressOnlyInternetGateways: []types.EgressOnlyInternetGateway{
			{
				Attachments: []types.InternetGatewayAttachment{
					{
						State: types.AttachmentStatusAttached,
						VpcId: adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
					},
				},
				EgressOnlyInternetGatewayId: adapterhelpers.PtrString("eigw-0ff50f360e066777a"),
			},
		},
	}

	items, err := egressOnlyInternetGatewayOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2EgressOnlyInternetGatewayAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2EgressOnlyInternetGatewayAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
