package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestGlobalNetworkOutputMapper(t *testing.T) {
	output := networkmanager.DescribeGlobalNetworksOutput{
		GlobalNetworks: []types.GlobalNetwork{
			{
				GlobalNetworkArn: adapterhelpers.PtrString("arn:aws:networkmanager:eu-west-2:052392120703:networkmanager/global-network/default"),
				GlobalNetworkId:  adapterhelpers.PtrString("default"),
			},
		},
	}

	items, err := globalNetworkOutputMapper(context.Background(), &networkmanager.Client{}, "foo", nil, &output)

	if err != nil {
		t.Error(err)
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
			ExpectedType:   "networkmanager-site",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-transit-gateway-registration",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-connect-peer-association",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-transit-gateway-connect-peer-association",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-network-resource",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-network-resource-relationship",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-link",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-device",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "networkmanager-connection",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}
