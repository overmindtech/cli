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

func TestInternetGatewayInputMapperGet(t *testing.T) {
	input, err := internetGatewayInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.InternetGatewayIds) != 1 {
		t.Fatalf("expected 1 InternetGateway ID, got %v", len(input.InternetGatewayIds))
	}

	if input.InternetGatewayIds[0] != "bar" {
		t.Errorf("expected InternetGateway ID to be bar, got %v", input.InternetGatewayIds[0])
	}
}

func TestInternetGatewayInputMapperList(t *testing.T) {
	input, err := internetGatewayInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.InternetGatewayIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestInternetGatewayOutputMapper(t *testing.T) {
	output := &ec2.DescribeInternetGatewaysOutput{
		InternetGateways: []types.InternetGateway{
			{
				Attachments: []types.InternetGatewayAttachment{
					{
						State: types.AttachmentStatusAttached,
						VpcId: adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
					},
				},
				InternetGatewayId: adapterhelpers.PtrString("igw-03809416c9e2fcb66"),
				OwnerId:           adapterhelpers.PtrString("052392120703"),
				Tags: []types.Tag{
					{
						Key:   adapterhelpers.PtrString("Name"),
						Value: adapterhelpers.PtrString("test"),
					},
				},
			},
		},
	}

	items, err := internetGatewayOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2InternetGatewayAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2InternetGatewayAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
