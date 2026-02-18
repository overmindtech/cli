package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestParsePropagationQuery(t *testing.T) {
	rt, att, err := parsePropagationQuery("tgw-rtb-1|tgw-attach-2")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || att != "tgw-attach-2" {
		t.Errorf("expected tgw-rtb-1, tgw-attach-2 got %q, %q", rt, att)
	}
	// Terraform uses underscore as separator
	rt, att, err = parsePropagationQuery("tgw-rtb-1_tgw-attach-2")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || att != "tgw-attach-2" {
		t.Errorf("expected tgw-rtb-1, tgw-attach-2 (underscore) got %q, %q", rt, att)
	}
	_, _, err = parsePropagationQuery("bad")
	if err == nil {
		t.Error("expected error for bad query")
	}
}

func TestTransitGatewayRouteTablePropagationItemMapper(t *testing.T) {
	item := &transitGatewayRouteTablePropagationItem{
		RouteTableID: "tgw-rtb-123",
		Propagation: types.TransitGatewayRouteTablePropagation{
			TransitGatewayAttachmentId: PtrString("tgw-attach-456"),
			ResourceId:                 PtrString("vpc-abc"),
			ResourceType:               types.TransitGatewayAttachmentResourceTypeVpc,
			State:                      types.TransitGatewayPropagationStateEnabled,
		},
	}
	sdpItem, err := transitGatewayRouteTablePropagationItemMapper("", "account|region", item)
	if err != nil {
		t.Fatal(err)
	}
	if err := sdpItem.Validate(); err != nil {
		t.Error(err)
	}
	if sdpItem.GetType() != "ec2-transit-gateway-route-table-propagation" {
		t.Errorf("unexpected type %s", sdpItem.GetType())
	}
}

func TestNewEC2TransitGatewayRouteTablePropagationAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)
	adapter := NewEC2TransitGatewayRouteTablePropagationAdapter(client, account, region, sdpcache.NewNoOpCache())
	if err := adapter.Validate(); err != nil {
		t.Fatal(err)
	}
}
