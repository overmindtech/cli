package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestSiteToSiteVpnAttachmentOutputMapper(t *testing.T) {
	scope := "123456789012.eu-west-2"
	tests := []struct {
		name           string
		item           *types.SiteToSiteVpnAttachment
		expectedHealth sdp.Health
		expectedAttr   string
		tests          adapterhelpers.QueryTests
	}{
		{
			name: "ok",
			item: &types.SiteToSiteVpnAttachment{
				Attachment: &types.Attachment{
					AttachmentId:  adapterhelpers.PtrString("stsa-1"),
					CoreNetworkId: adapterhelpers.PtrString("cn-1"),
					State:         types.AttachmentStateAvailable,
				},
				VpnConnectionArn: adapterhelpers.PtrString("arn:aws:ec2:us-west-2:123456789012:vpn-connection/vpn-1234"),
			},
			expectedHealth: sdp.Health_HEALTH_OK,
			expectedAttr:   "stsa-1",
			tests: adapterhelpers.QueryTests{
				{
					ExpectedType:   "networkmanager-core-network",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "cn-1",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   "ec2-vpn-connection",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "arn:aws:ec2:us-west-2:123456789012:vpn-connection/vpn-1234",
					ExpectedScope:  scope,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := siteToSiteVpnAttachmentItemMapper("", scope, tt.item)
			if err != nil {
				t.Error(err)
			}

			if item.UniqueAttributeValue() != tt.expectedAttr {
				t.Fatalf("want %s, got %s", tt.expectedAttr, item.UniqueAttributeValue())
			}

			if tt.expectedHealth != item.GetHealth() {
				t.Fatalf("want %d, got %d", tt.expectedHealth, item.GetHealth())
			}

			tt.tests.Execute(t, item)
		})
	}
}
