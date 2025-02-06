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

func TestDirectConnectGatewayAssociationOutputMapper_Health_OK(t *testing.T) {
	output := &directconnect.DescribeDirectConnectGatewayAssociationsOutput{
		DirectConnectGatewayAssociations: []types.DirectConnectGatewayAssociation{
			{
				AssociationState:           types.DirectConnectGatewayAssociationStateAssociating,
				AssociationId:              adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52cexample"),
				VirtualGatewayOwnerAccount: adapterhelpers.PtrString("123456789012"),
				DirectConnectGatewayId:     adapterhelpers.PtrString("5f294f92-bafb-4011-916d-9b0bexample"),
				VirtualGatewayId:           adapterhelpers.PtrString("vgw-6efe725e"),
			},
		},
	}

	items, err := directConnectGatewayAssociationOutputMapper(context.Background(), nil, "foo", nil, output)
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

	if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Fatalf("expected health to be OK, got: %v", item.GetHealth())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "directconnect-direct-connect-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "5f294f92-bafb-4011-916d-9b0bexample",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "directconnect-virtual-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vgw-6efe725e",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestDirectConnectGatewayAssociationOutputMapper_Health_Error(t *testing.T) {
	output := &directconnect.DescribeDirectConnectGatewayAssociationsOutput{
		DirectConnectGatewayAssociations: []types.DirectConnectGatewayAssociation{
			{
				AssociationState:           types.DirectConnectGatewayAssociationStateAssociating,
				AssociationId:              adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52cexample"),
				VirtualGatewayOwnerAccount: adapterhelpers.PtrString("123456789012"),
				DirectConnectGatewayId:     adapterhelpers.PtrString("5f294f92-bafb-4011-916d-9b0bexample"),
				VirtualGatewayId:           adapterhelpers.PtrString("vgw-6efe725e"),
				StateChangeError:           adapterhelpers.PtrString("something went wrong"),
			},
		},
	}

	items, err := directConnectGatewayAssociationOutputMapper(context.Background(), nil, "foo", nil, output)
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

	if item.GetHealth() != sdp.Health_HEALTH_ERROR {
		t.Fatalf("expected health to be ERROR, got: %v", item.GetHealth())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "directconnect-direct-connect-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "5f294f92-bafb-4011-916d-9b0bexample",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "directconnect-virtual-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vgw-6efe725e",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDirectConnectGatewayAssociationAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectGatewayAssociationAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
