package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestLinkOutputMapper(t *testing.T) {
	output := networkmanager.GetLinksOutput{
		Links: []types.Link{
			{
				LinkId:          adapterhelpers.PtrString("link-1"),
				GlobalNetworkId: adapterhelpers.PtrString("default"),
				SiteId:          adapterhelpers.PtrString("site-1"),
				LinkArn:         adapterhelpers.PtrString("arn:aws:networkmanager:us-west-2:123456789012:link/link-1"),
			},
		},
	}
	scope := "123456789012.eu-west-2"
	items, err := linkOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetLinksInput{}, &output)

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

	// Ensure unique attribute
	err = item.Validate()

	if err != nil {
		t.Error(err)
	}

	if item.UniqueAttributeValue() != "default|link-1" {
		t.Fatalf("expected default|link-1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-site",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|site-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-network-resource-relationship",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default|arn:aws:networkmanager:us-west-2:123456789012:link/link-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link-association",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|link|link-1",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}
