package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestDirectConnectGatewayAssociationProposalOutputMapper(t *testing.T) {
	output := &directconnect.DescribeDirectConnectGatewayAssociationProposalsOutput{
		DirectConnectGatewayAssociationProposals: []types.DirectConnectGatewayAssociationProposal{
			{
				ProposalId:                       PtrString("c2ede9b4-bbc6-4d33-923c-bc4feEXAMPLE"),
				DirectConnectGatewayId:           PtrString("5f294f92-bafb-4011-916d-9b0bexample"),
				DirectConnectGatewayOwnerAccount: PtrString("123456789012"),
				ProposalState:                    types.DirectConnectGatewayAssociationProposalStateRequested,
				AssociatedGateway: &types.AssociatedGateway{
					Id:           PtrString("tgw-02f776b1a7EXAMPLE"),
					Type:         types.GatewayTypeTransitGateway,
					OwnerAccount: PtrString("111122223333"),
					Region:       PtrString("us-east-1"),
				},
				ExistingAllowedPrefixesToDirectConnectGateway: []types.RouteFilterPrefix{
					{
						Cidr: PtrString("192.168.2.0/30"),
					},
					{
						Cidr: PtrString("192.168.1.0/30"),
					},
				},
				RequestedAllowedPrefixesToDirectConnectGateway: []types.RouteFilterPrefix{
					{
						Cidr: PtrString("192.168.1.0/30"),
					},
				},
			},
		},
	}

	items, err := directConnectGatewayAssociationProposalOutputMapper(context.Background(), nil, "foo", nil, output)
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

	tests := QueryTests{
		{
			ExpectedType:   "directconnect-direct-connect-gateway-association",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  fmt.Sprintf("%s/%s", "5f294f92-bafb-4011-916d-9b0bexample", "tgw-02f776b1a7EXAMPLE"),
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDirectConnectGatewayAssociationProposalAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectGatewayAssociationProposalAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
