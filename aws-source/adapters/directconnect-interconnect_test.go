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

func TestInterconnectOutputMapper(t *testing.T) {
	output := &directconnect.DescribeInterconnectsOutput{
		Interconnects: []types.Interconnect{
			{
				AwsDeviceV2:          adapterhelpers.PtrString("EqDC2-123h49s71dabc"),
				AwsLogicalDeviceId:   adapterhelpers.PtrString("device-1"),
				Bandwidth:            adapterhelpers.PtrString("1Gbps"),
				HasLogicalRedundancy: types.HasLogicalRedundancyUnknown,
				InterconnectId:       adapterhelpers.PtrString("dxcon-fguhmqlc"),
				InterconnectName:     adapterhelpers.PtrString("interconnect-1"),
				InterconnectState:    types.InterconnectStateAvailable,
				JumboFrameCapable:    adapterhelpers.PtrBool(true),
				LagId:                adapterhelpers.PtrString("dxlag-ffrz71kw"),
				LoaIssueTime:         adapterhelpers.PtrTime(time.Now()),
				Location:             adapterhelpers.PtrString("EqDC2"),
				Region:               adapterhelpers.PtrString("us-east-1"),
				ProviderName:         adapterhelpers.PtrString("provider-1"),
				Tags: []types.Tag{
					{
						Key:   adapterhelpers.PtrString("foo"),
						Value: adapterhelpers.PtrString("bar"),
					},
				},
			},
		},
	}

	items, err := interconnectOutputMapper(context.Background(), nil, "foo", nil, output)
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

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "directconnect-lag",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dxlag-ffrz71kw",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-location",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "EqDC2",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-loa",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dxcon-fguhmqlc",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "directconnect-hosted-connection",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "dxcon-fguhmqlc",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestInterconnectHealth(t *testing.T) {
	cases := []struct {
		state  types.InterconnectState
		health sdp.Health
	}{
		{
			state:  types.InterconnectStateRequested,
			health: sdp.Health_HEALTH_PENDING,
		},
		{
			state:  types.InterconnectStatePending,
			health: sdp.Health_HEALTH_PENDING,
		},
		{
			state:  types.InterconnectStateAvailable,
			health: sdp.Health_HEALTH_OK,
		},
		{
			state:  types.InterconnectStateDown,
			health: sdp.Health_HEALTH_ERROR,
		},
		{
			state:  types.InterconnectStateDeleting,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
		{
			state:  types.InterconnectStateDeleted,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
		{
			state:  types.InterconnectStateUnknown,
			health: sdp.Health_HEALTH_UNKNOWN,
		},
	}

	for _, c := range cases {
		output := &directconnect.DescribeInterconnectsOutput{
			Interconnects: []types.Interconnect{
				{
					InterconnectState: c.state,
					LagId:             adapterhelpers.PtrString("dxlag-fgsu9erb"),
				},
			},
		}

		items, err := interconnectOutputMapper(context.Background(), nil, "foo", nil, output)
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

func TestNewDirectConnectInterconnectAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectInterconnectAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
		// Listing these in our test account gives "An error occurred
		// (DirectConnectClientException) when calling the DescribeInterconnects
		// operation: Account [NUMBER] is not an authorized Direct Connect
		// partner in eu-west-2."
		//
		// Skipping tests for now
		SkipList: true,
	}

	test.Run(t)
}
