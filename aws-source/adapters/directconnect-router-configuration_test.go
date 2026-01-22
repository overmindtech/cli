package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestRouterConfigurationOutputMapper(t *testing.T) {
	output := &directconnect.DescribeRouterConfigurationOutput{
		CustomerRouterConfig: PtrString("some config"),
		Router: &types.RouterType{
			Platform:                  PtrString("2900 Series Routers"),
			RouterTypeIdentifier:      PtrString("CiscoSystemsInc-2900SeriesRouters-IOS124"),
			Software:                  PtrString("IOS 12.4+"),
			Vendor:                    PtrString("Cisco Systems, Inc."),
			XsltTemplateName:          PtrString("customer-router-cisco-generic.xslt"),
			XsltTemplateNameForMacSec: PtrString(""),
		},
		VirtualInterfaceId:   PtrString("dxvif-ffhhk74f"),
		VirtualInterfaceName: PtrString("PrivateVirtualInterface"),
	}

	items, err := routerConfigurationOutputMapper(context.Background(), nil, "foo", nil, output)
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
			ExpectedType:   "directconnect-virtual-interface",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "dxvif-ffhhk74f",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDirectConnectRouterConfigurationAdapter(t *testing.T) {
	client, account, region := directconnectGetAutoConfig(t)

	adapter := NewDirectConnectRouterConfigurationAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
