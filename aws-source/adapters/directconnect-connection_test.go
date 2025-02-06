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

func TestDirectconnectConnectionOutputMapper(t *testing.T) {
	output := &directconnect.DescribeConnectionsOutput{
		Connections: []types.Connection{
			{
				AwsDeviceV2:          adapterhelpers.PtrString("EqDC2-123h49s71dabc"),
				AwsLogicalDeviceId:   adapterhelpers.PtrString("device-1"),
				Bandwidth:            adapterhelpers.PtrString("1Gbps"),
				ConnectionId:         adapterhelpers.PtrString("dxcon-fguhmqlc"),
				ConnectionName:       adapterhelpers.PtrString("My_Connection"),
				ConnectionState:      "down",
				EncryptionMode:       adapterhelpers.PtrString("must_encrypt"),
				HasLogicalRedundancy: "unknown",
				JumboFrameCapable:    adapterhelpers.PtrBool(true),
				LagId:                adapterhelpers.PtrString("dxlag-ffrz71kw"),
				LoaIssueTime:         adapterhelpers.PtrTime(time.Now()),
				Location:             adapterhelpers.PtrString("EqDC2"),
				Region:               adapterhelpers.PtrString("us-east-1"),
				ProviderName:         adapterhelpers.PtrString("provider-1"),
				OwnerAccount:         adapterhelpers.PtrString("123456789012"),
				PartnerName:          adapterhelpers.PtrString("partner-1"),
				Tags: []types.Tag{
					{
						Key:   adapterhelpers.PtrString("foo"),
						Value: adapterhelpers.PtrString("bar"),
					},
				},
			},
		},
	}

	items, err := directconnectConnectionOutputMapper(context.Background(), nil, "foo", nil, output)
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
			ExpectedType:   "directconnect-virtual-interface",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "dxcon-fguhmqlc",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDirectConnectConnectionAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectConnectionAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
