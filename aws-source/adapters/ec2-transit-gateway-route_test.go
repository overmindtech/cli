package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestTransitGatewayRouteDestination(t *testing.T) {
	if transitGatewayRouteDestination(&types.TransitGatewayRoute{DestinationCidrBlock: new("10.0.0.0/16")}) != "10.0.0.0/16" {
		t.Error("expected CIDR destination")
	}
	if transitGatewayRouteDestination(&types.TransitGatewayRoute{PrefixListId: new("pl-123")}) != "pl:pl-123" {
		t.Error("expected prefix list destination")
	}
}

func TestParseRouteQuery(t *testing.T) {
	rt, dest, err := parseRouteQuery("tgw-rtb-1|10.0.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || dest != "10.0.0.0/16" {
		t.Errorf("expected tgw-rtb-1, 10.0.0.0/16 got %q, %q", rt, dest)
	}
	// Terraform uses underscore as separator
	rt, dest, err = parseRouteQuery("tgw-rtb-1_10.0.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || dest != "10.0.0.0/16" {
		t.Errorf("expected tgw-rtb-1, 10.0.0.0/16 (underscore) got %q, %q", rt, dest)
	}
	_, _, err = parseRouteQuery("bad")
	if err == nil {
		t.Error("expected error for bad query")
	}
}

func TestTransitGatewayRouteItemMapper(t *testing.T) {
	item := &transitGatewayRouteItem{
		RouteTableID: "tgw-rtb-123",
		Route: types.TransitGatewayRoute{
			DestinationCidrBlock: new("10.0.0.0/16"),
			State:                types.TransitGatewayRouteStateActive,
			Type:                 types.TransitGatewayRouteTypeStatic,
		},
	}
	sdpItem, err := transitGatewayRouteItemMapper("", "account|region", item)
	if err != nil {
		t.Fatal(err)
	}
	if err := sdpItem.Validate(); err != nil {
		t.Error(err)
	}
	if sdpItem.GetType() != "ec2-transit-gateway-route" {
		t.Errorf("unexpected type %s", sdpItem.GetType())
	}
}

func TestNewEC2TransitGatewayRouteAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)
	adapter := NewEC2TransitGatewayRouteAdapter(client, account, region, sdpcache.NewNoOpCache())
	if err := adapter.Validate(); err != nil {
		t.Fatal(err)
	}
}
