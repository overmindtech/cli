package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
)

func TestLagHealth(t *testing.T) {
	cases := []struct {
		state  types.LagState
		health sdp.Health
	}{
		{
			state:  types.LagStateRequested,
			health: sdp.Health_HEALTH_PENDING,
		},
		{
			state:  types.LagStatePending,
			health: sdp.Health_HEALTH_PENDING,
		},
		{
			state:  types.LagStateAvailable,
			health: sdp.Health_HEALTH_OK,
		},
		{
			state:  types.LagStateDown,
			health: sdp.Health_HEALTH_ERROR,
		},
		{
			state:  types.LagStateDeleting,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
		{
			state:  types.LagStateDeleted,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
		{
			state:  types.LagStateUnknown,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
	}

	for _, c := range cases {
		output := &directconnect.DescribeLagsOutput{
			Lags: []types.Lag{
				{
					LagState: c.state,
					LagId:    adapterhelpers.PtrString("dxlag-fgsu9erb"),
				},
			},
		}

		items, err := lagOutputMapper(context.Background(), nil, "foo", nil, output)
		if err != nil {
			t.Fatal(err)
		}

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %v", len(items))
		}

		item := items[0]

		if item.GetHealth() != c.health {
			t.Errorf("expected health to be %v, got: %v", c.health, item.GetHealth())
		}
	}
}

func TestLagOutputMapper(t *testing.T) {
	output := &directconnect.DescribeLagsOutput{
		Lags: []types.Lag{
			{
				AwsDeviceV2:         adapterhelpers.PtrString("EqDC2-19y7z3m17xpuz"),
				NumberOfConnections: int32(2),
				LagState:            types.LagStateAvailable,
				OwnerAccount:        adapterhelpers.PtrString("123456789012"),
				LagName:             adapterhelpers.PtrString("DA-LAG"),
				Connections: []types.Connection{
					{
						OwnerAccount:    adapterhelpers.PtrString("123456789012"),
						ConnectionId:    adapterhelpers.PtrString("dxcon-ffnikghc"),
						LagId:           adapterhelpers.PtrString("dxlag-fgsu9erb"),
						ConnectionState: "requested",
						Bandwidth:       adapterhelpers.PtrString("10Gbps"),
						Location:        adapterhelpers.PtrString("EqDC2"),
						ConnectionName:  adapterhelpers.PtrString("Requested Connection 1 for Lag dxlag-fgsu9erb"),
						Region:          adapterhelpers.PtrString("us-east-1"),
					},
					{
						OwnerAccount:    adapterhelpers.PtrString("123456789012"),
						ConnectionId:    adapterhelpers.PtrString("dxcon-fglgbdea"),
						LagId:           adapterhelpers.PtrString("dxlag-fgsu9erb"),
						ConnectionState: "requested",
						Bandwidth:       adapterhelpers.PtrString("10Gbps"),
						Location:        adapterhelpers.PtrString("EqDC2"),
						ConnectionName:  adapterhelpers.PtrString("Requested Connection 2 for Lag dxlag-fgsu9erb"),
						Region:          adapterhelpers.PtrString("us-east-1"),
					},
				},
				LagId:                adapterhelpers.PtrString("dxlag-fgsu9erb"),
				MinimumLinks:         int32(0),
				ConnectionsBandwidth: adapterhelpers.PtrString("10Gbps"),
				Region:               adapterhelpers.PtrString("us-east-1"),
				Location:             adapterhelpers.PtrString("EqDC2"),
			},
		},
	}

	items, err := lagOutputMapper(context.Background(), nil, "foo", nil, output)
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
			ExpectedType:   "directconnect-connection",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dxcon-ffnikghc",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-connection",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dxcon-fglgbdea",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-location",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "EqDC2",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-hosted-connection",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "dxlag-fgsu9erb",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDirectConnectLagAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectLagAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
