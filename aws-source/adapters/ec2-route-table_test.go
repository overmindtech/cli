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

func TestRouteTableInputMapperGet(t *testing.T) {
	input, err := routeTableInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.RouteTableIds) != 1 {
		t.Fatalf("expected 1 RouteTable ID, got %v", len(input.RouteTableIds))
	}

	if input.RouteTableIds[0] != "bar" {
		t.Errorf("expected RouteTable ID to be bar, got %v", input.RouteTableIds[0])
	}
}

func TestRouteTableInputMapperList(t *testing.T) {
	input, err := routeTableInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.RouteTableIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestRouteTableOutputMapper(t *testing.T) {
	output := &ec2.DescribeRouteTablesOutput{
		RouteTables: []types.RouteTable{
			{
				Associations: []types.RouteTableAssociation{
					{
						Main:                    adapterhelpers.PtrBool(false),
						RouteTableAssociationId: adapterhelpers.PtrString("rtbassoc-0aa1442039abff3db"),
						RouteTableId:            adapterhelpers.PtrString("rtb-00b1197fa95a6b35f"),
						SubnetId:                adapterhelpers.PtrString("subnet-06c0dea0437180c61"),
						GatewayId:               adapterhelpers.PtrString("ID"),
						AssociationState: &types.RouteTableAssociationState{
							State: types.RouteTableAssociationStateCodeAssociated,
						},
					},
				},
				PropagatingVgws: []types.PropagatingVgw{
					{
						GatewayId: adapterhelpers.PtrString("goo"),
					},
				},
				RouteTableId: adapterhelpers.PtrString("rtb-00b1197fa95a6b35f"),
				Routes: []types.Route{
					{
						DestinationCidrBlock: adapterhelpers.PtrString("172.31.0.0/16"),
						GatewayId:            adapterhelpers.PtrString("igw-12345"),
						Origin:               types.RouteOriginCreateRouteTable,
						State:                types.RouteStateActive,
					},
					{
						DestinationPrefixListId:     adapterhelpers.PtrString("pl-7ca54015"),
						GatewayId:                   adapterhelpers.PtrString("vpce-09fcbac4dcf142db3"),
						Origin:                      types.RouteOriginCreateRoute,
						State:                       types.RouteStateActive,
						CarrierGatewayId:            adapterhelpers.PtrString("id"),
						EgressOnlyInternetGatewayId: adapterhelpers.PtrString("id"),
						InstanceId:                  adapterhelpers.PtrString("id"),
						InstanceOwnerId:             adapterhelpers.PtrString("id"),
						LocalGatewayId:              adapterhelpers.PtrString("id"),
						NatGatewayId:                adapterhelpers.PtrString("id"),
						NetworkInterfaceId:          adapterhelpers.PtrString("id"),
						TransitGatewayId:            adapterhelpers.PtrString("id"),
						VpcPeeringConnectionId:      adapterhelpers.PtrString("id"),
					},
				},
				VpcId:   adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
				OwnerId: adapterhelpers.PtrString("052392120703"),
			},
		},
	}

	items, err := routeTableOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet-06c0dea0437180c61",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-internet-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "ID",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-carrier-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-egress-only-internet-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-local-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-nat-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-network-interface",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-transit-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc-peering-connection",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc-endpoint",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpce-09fcbac4dcf142db3",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-internet-gateway",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "igw-12345",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2RouteTableAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2RouteTableAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
