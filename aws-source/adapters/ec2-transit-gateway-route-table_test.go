package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestTransitGatewayRouteTableInputMapperGet(t *testing.T) {
	input, err := transitGatewayRouteTableInputMapperGet("foo", "tgw-rtb-123")

	if err != nil {
		t.Error(err)
	}

	if len(input.TransitGatewayRouteTableIds) != 1 {
		t.Fatalf("expected 1 TransitGatewayRouteTable ID, got %v", len(input.TransitGatewayRouteTableIds))
	}

	if input.TransitGatewayRouteTableIds[0] != "tgw-rtb-123" {
		t.Errorf("expected TransitGatewayRouteTable ID to be tgw-rtb-123, got %v", input.TransitGatewayRouteTableIds[0])
	}
}

func TestTransitGatewayRouteTableInputMapperList(t *testing.T) {
	input, err := transitGatewayRouteTableInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.TransitGatewayRouteTableIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestTransitGatewayRouteTableOutputMapper(t *testing.T) {
	output := &ec2.DescribeTransitGatewayRouteTablesOutput{
		TransitGatewayRouteTables: []types.TransitGatewayRouteTable{
			{
				TransitGatewayRouteTableId:   PtrString("tgw-rtb-0123456789abcdef0"),
				TransitGatewayId:             PtrString("tgw-0abc123"),
				State:                        types.TransitGatewayRouteTableStateAvailable,
				DefaultAssociationRouteTable: PtrBool(false),
				DefaultPropagationRouteTable: PtrBool(false),
				Tags: []types.Tag{
					{Key: PtrString("Name"), Value: PtrString("my-route-table")},
				},
			},
		},
	}

	items, err := transitGatewayRouteTableOutputMapper(context.Background(), nil, "foo", nil, output)

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

	if items[0].GetUniqueAttribute() != "TransitGatewayRouteTableId" {
		t.Errorf("expected UniqueAttribute TransitGatewayRouteTableId, got %v", items[0].GetUniqueAttribute())
	}

	// Should link to ec2-transit-gateway and to associations, propagations, routes (Search by route table ID)
	links := items[0].GetLinkedItemQueries()
	if len(links) != 4 {
		t.Fatalf("expected 4 linked item queries (ec2-transit-gateway + 3 Search), got %v", len(links))
	}
	if links[0].GetQuery().GetType() != "ec2-transit-gateway" {
		t.Errorf("expected first link type ec2-transit-gateway, got %v", links[0].GetQuery().GetType())
	}
	searchTypes := map[string]bool{}
	for _, l := range links[1:] {
		if l.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
			t.Errorf("expected Search method for link %s", l.GetQuery().GetType())
		}
		searchTypes[l.GetQuery().GetType()] = true
	}
	for _, want := range []string{"ec2-transit-gateway-route-table-association", "ec2-transit-gateway-route-table-propagation", "ec2-transit-gateway-route"} {
		if !searchTypes[want] {
			t.Errorf("expected Search link to %s", want)
		}
	}
}

func TestNewEC2TransitGatewayRouteTableAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2TransitGatewayRouteTableAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
