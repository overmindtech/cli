package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestDirectconnectConnectionOutputMapper(t *testing.T) {
	output := &directconnect.DescribeConnectionsOutput{
		Connections: []types.Connection{
			{
				AwsDeviceV2:          PtrString("EqDC2-123h49s71dabc"),
				AwsLogicalDeviceId:   PtrString("device-1"),
				Bandwidth:            PtrString("1Gbps"),
				ConnectionId:         PtrString("dxcon-fguhmqlc"),
				ConnectionName:       PtrString("My_Connection"),
				ConnectionState:      "down",
				EncryptionMode:       PtrString("must_encrypt"),
				HasLogicalRedundancy: "unknown",
				JumboFrameCapable:    PtrBool(true),
				LagId:                PtrString("dxlag-ffrz71kw"),
				LoaIssueTime:         PtrTime(time.Now()),
				Location:             PtrString("EqDC2"),
				Region:               PtrString("us-east-1"),
				ProviderName:         PtrString("provider-1"),
				OwnerAccount:         PtrString("123456789012"),
				PartnerName:          PtrString("partner-1"),
				Tags: []types.Tag{
					{
						Key:   PtrString("foo"),
						Value: PtrString("bar"),
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

	tests := QueryTests{
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

	adapter := NewDirectConnectConnectionAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
