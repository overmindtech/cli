package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestVPCAttachmentItemMapper(t *testing.T) {
	input := types.VpcAttachment{
		Attachment: &types.Attachment{
			AttachmentId:  adapterhelpers.PtrString("attachment1"),
			CoreNetworkId: adapterhelpers.PtrString("corenetwork1"),
		},
	}
	scope := "123456789012.eu-west-2"
	item, err := vpcAttachmentItemMapper("", scope, &input)

	if err != nil {
		t.Error(err)
	}
	if err := item.Validate(); err != nil {
		t.Error(err)
	}

	// Ensure unique attribute
	if item.UniqueAttributeValue() != "attachment1" {
		t.Fatalf("expected %v, got %v", "attachment1", item.UniqueAttributeValue())
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "networkmanager-core-network",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "corenetwork1",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}
