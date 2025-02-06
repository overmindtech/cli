package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestSiteOutputMapper(t *testing.T) {
	output := networkmanager.GetSitesOutput{
		Sites: []types.Site{
			{
				SiteId:          adapterhelpers.PtrString("site1"),
				GlobalNetworkId: adapterhelpers.PtrString("default"),
			},
		},
	}
	scope := "123456789012.eu-west-2"
	items, err := siteOutputMapper(context.Background(), &networkmanager.Client{}, scope, &networkmanager.GetSitesInput{}, &output)

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

	if item.UniqueAttributeValue() != "default|site1" {
		t.Fatalf("expected default|site1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-global-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-link",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|site1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-device",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "default|site1",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}
