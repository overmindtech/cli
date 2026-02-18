package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestParseAssociationQuery(t *testing.T) {
	rt, att, err := parseAssociationQuery("tgw-rtb-1|tgw-attach-2")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || att != "tgw-attach-2" {
		t.Errorf("expected tgw-rtb-1, tgw-attach-2 got %q, %q", rt, att)
	}
	// Terraform uses underscore as separator
	rt, att, err = parseAssociationQuery("tgw-rtb-1_tgw-attach-2")
	if err != nil {
		t.Fatal(err)
	}
	if rt != "tgw-rtb-1" || att != "tgw-attach-2" {
		t.Errorf("expected tgw-rtb-1, tgw-attach-2 (underscore) got %q, %q", rt, att)
	}
	_, _, err = parseAssociationQuery("bad")
	if err == nil {
		t.Error("expected error for bad query")
	}
}

func TestTransitGatewayRouteTableAssociationItemMapper(t *testing.T) {
	item := &transitGatewayRouteTableAssociationItem{
		RouteTableID: "tgw-rtb-123",
		Association: types.TransitGatewayRouteTableAssociation{
			TransitGatewayAttachmentId: PtrString("tgw-attach-456"),
			ResourceId:                 PtrString("vpc-abc"),
			ResourceType:               types.TransitGatewayAttachmentResourceTypeVpc,
			State:                      types.TransitGatewayAssociationStateAssociated,
		},
	}
	sdpItem, err := transitGatewayRouteTableAssociationItemMapper("", "account|region", item)
	if err != nil {
		t.Fatal(err)
	}
	if err := sdpItem.Validate(); err != nil {
		t.Error(err)
	}
	if sdpItem.GetType() != "ec2-transit-gateway-route-table-association" {
		t.Errorf("unexpected type %s", sdpItem.GetType())
	}
	uv, _ := sdpItem.GetAttributes().Get("TransitGatewayRouteTableIdWithTransitGatewayAttachmentId")
	if uv != "tgw-rtb-123|tgw-attach-456" {
		t.Errorf("unexpected unique value %v", uv)
	}
}

func TestNewEC2TransitGatewayRouteTableAssociationAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)
	adapter := NewEC2TransitGatewayRouteTableAssociationAdapter(client, account, region, sdpcache.NewNoOpCache())
	if err := adapter.Validate(); err != nil {
		t.Fatal(err)
	}
}
