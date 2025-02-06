package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestVirtualGatewayOutputMapper(t *testing.T) {
	output := &directconnect.DescribeVirtualGatewaysOutput{
		VirtualGateways: []types.VirtualGateway{
			{
				VirtualGatewayId:    adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52cexample"),
				VirtualGatewayState: adapterhelpers.PtrString("available"),
			},
		},
	}

	items, err := virtualGatewayOutputMapper(context.Background(), nil, "foo", nil, output)
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
}

func TestNewDirectConnectVirtualGatewayAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectVirtualGatewayAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
