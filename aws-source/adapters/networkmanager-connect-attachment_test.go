package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestConnectAttachmentItemMapper(t *testing.T) {

	scope := "123456789012.eu-west-2"
	item, err := connectAttachmentItemMapper("", scope, &types.ConnectAttachment{
		Attachment: &types.Attachment{
			AttachmentId:   adapterhelpers.PtrString("att-1"),
			CoreNetworkId:  adapterhelpers.PtrString("cn-1"),
			CoreNetworkArn: adapterhelpers.PtrString("arn:aws:networkmanager:eu-west-2:123456789012:core-network/cn-1"),
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Ensure unique attribute
	err = item.Validate()
	if err != nil {
		t.Error(err)
	}

	if item.UniqueAttributeValue() != "att-1" {
		t.Fatalf("expected att-1, got %v", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-core-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cn-1",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "networkmanager-core-network",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:networkmanager:eu-west-2:123456789012:core-network/cn-1",
			ExpectedScope:  "123456789012.eu-west-2",
		},
	}

	tests.Execute(t, item)
}
