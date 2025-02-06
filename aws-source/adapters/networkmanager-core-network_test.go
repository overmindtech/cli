package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (n NetworkManagerTestClient) GetCoreNetwork(ctx context.Context, params *networkmanager.GetCoreNetworkInput, optFns ...func(*networkmanager.Options)) (*networkmanager.GetCoreNetworkOutput, error) {
	return &networkmanager.GetCoreNetworkOutput{
		CoreNetwork: &types.CoreNetwork{
			CoreNetworkArn:  adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:core-network/cn-1"),
			CoreNetworkId:   adapterhelpers.PtrString("cn-1"),
			GlobalNetworkId: adapterhelpers.PtrString("default"),
			Description:     adapterhelpers.PtrString("core network description"),
			State:           types.CoreNetworkStateAvailable,
			Edges: []types.CoreNetworkEdge{
				{
					Asn:          adapterhelpers.PtrInt64(64512), // link
					EdgeLocation: adapterhelpers.PtrString("us-west-2"),
				},
			},
			Segments: []types.CoreNetworkSegment{
				{
					EdgeLocations: []string{"us-west-2"},
					Name:          adapterhelpers.PtrString("segment-1"),
				},
			},
		},
	}, nil
}

func (n NetworkManagerTestClient) ListCoreNetworks(context.Context, *networkmanager.ListCoreNetworksInput, ...func(*networkmanager.Options)) (*networkmanager.ListCoreNetworksOutput, error) {
	return nil, nil
}

func TestCoreNetworkItemMapper(t *testing.T) {
	item, err := coreNetworkGetFunc(context.Background(), NetworkManagerTestClient{}, "test", &networkmanager.GetCoreNetworkInput{})
	if err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Error(err)
	}

	// Ensure unique attribute
	err = item.Validate()
	if err != nil {
		t.Error(err)
	}

	if item.UniqueAttributeValue() != "cn-1" {
		t.Fatalf("expected cn-1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "networkmanager-core-network-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cn-1",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "networkmanager-connect-peer",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "cn-1",
			ExpectedScope:  "test",
		},
		{
			ExpectedType:   "rdap-asn",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "64512",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}
