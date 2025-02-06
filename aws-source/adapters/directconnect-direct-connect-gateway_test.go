package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestDirectConnectGatewayOutputMapper_Health_OK(t *testing.T) {
	output := &directconnect.DescribeDirectConnectGatewaysOutput{
		DirectConnectGateways: []types.DirectConnectGateway{
			{
				AmazonSideAsn:             adapterhelpers.PtrInt64(64512),
				DirectConnectGatewayId:    adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52cexample"),
				OwnerAccount:              adapterhelpers.PtrString("123456789012"),
				DirectConnectGatewayName:  adapterhelpers.PtrString("DxGateway2"),
				DirectConnectGatewayState: types.DirectConnectGatewayStateAvailable,
			},
		},
	}

	items, err := directConnectGatewayOutputMapper(context.Background(), nil, "foo", nil, output)
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

	if items[0].GetHealth() != sdp.Health_HEALTH_OK {
		t.Fatalf("expected health to be OK, got: %v", items[0].GetHealth())
	}
}

func TestDirectConnectGatewayOutputMapper_Health_ERROR(t *testing.T) {
	output := &directconnect.DescribeDirectConnectGatewaysOutput{
		DirectConnectGateways: []types.DirectConnectGateway{
			{
				AmazonSideAsn:             adapterhelpers.PtrInt64(64512),
				DirectConnectGatewayId:    adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52cexample"),
				OwnerAccount:              adapterhelpers.PtrString("123456789012"),
				DirectConnectGatewayName:  adapterhelpers.PtrString("DxGateway2"),
				DirectConnectGatewayState: types.DirectConnectGatewayStateAvailable,
				StateChangeError:          adapterhelpers.PtrString("error"),
			},
		},
	}

	items, err := directConnectGatewayOutputMapper(context.Background(), nil, "foo", nil, output)
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

	if items[0].GetHealth() != sdp.Health_HEALTH_ERROR {
		t.Fatalf("expected health to be ERROR, got: %v", items[0].GetHealth())
	}
}

func TestNewDirectConnectGatewayAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectGatewayAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}

func Test_arn(t *testing.T) {
	tests := []struct {
		name      string
		region    string
		accountID string
		gatewayID string
		want      string
	}{
		{
			name:      "us-west-2",
			region:    "us-west-2",
			accountID: "123456789012",
			gatewayID: "cf68415c-f4ae-48f2-87a7-3b52cexample",
			want:      "arn:aws:directconnect:us-west-2:123456789012:dx-gateway/cf68415c-f4ae-48f2-87a7-3b52cexample",
		},
		{
			name:      "us-east-1",
			region:    "us-east-1",
			accountID: "123456789012",
			gatewayID: "cf68415c-f4ae-48f2-87a7-3b52cexample",
			want:      "arn:aws:directconnect:us-east-1:123456789012:dx-gateway/cf68415c-f4ae-48f2-87a7-3b52cexample",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := directconnectARN(tt.region, tt.accountID, tt.gatewayID); got != tt.want {
				t.Errorf("arn() = %v, want %v", got, tt.want)
			}
		})
	}
}
