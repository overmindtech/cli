package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestHostedConnectionOutputMapper(t *testing.T) {
	output := &directconnect.DescribeHostedConnectionsOutput{
		Connections: []types.Connection{
			{
				AwsDeviceV2:          new("EqDC2-123h49s71dabc"),
				AwsLogicalDeviceId:   new("device-1"),
				Bandwidth:            new("1Gbps"),
				ConnectionId:         new("dxcon-fguhmqlc"),
				ConnectionName:       new("My_Connection"),
				ConnectionState:      "down",
				EncryptionMode:       new("must_encrypt"),
				HasLogicalRedundancy: "unknown",
				JumboFrameCapable:    new(true),
				LagId:                new("dxlag-ffrz71kw"),
				LoaIssueTime:         new(time.Now()),
				Location:             new("EqDC2"),
				Region:               new("us-east-1"),
				ProviderName:         new("provider-1"),
				OwnerAccount:         new("123456789012"),
				PartnerName:          new("partner-1"),
				Tags: []types.Tag{
					{
						Key:   new("foo"),
						Value: new("bar"),
					},
				},
			},
		},
	}

	items, err := hostedConnectionOutputMapper(context.Background(), nil, "foo", nil, output)
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

func TestNewDirectConnectHostedConnectionAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectHostedConnectionAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
