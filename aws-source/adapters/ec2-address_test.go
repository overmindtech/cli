package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestAddressInputMapperGet(t *testing.T) {
	input, err := addressInputMapperGet("foo", "az-name")

	if err != nil {
		t.Error(err)
	}

	if len(input.PublicIps) != 1 {
		t.Fatalf("expected 1 Address, got %v", len(input.PublicIps))
	}

	if input.PublicIps[0] != "az-name" {
		t.Errorf("expected Address to be to be az-name, got %v", input.PublicIps[0])
	}
}

func TestAddressInputMapperList(t *testing.T) {
	input, err := addressInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.PublicIps) != 0 {
		t.Fatalf("expected 0 zone names, got %v", len(input.PublicIps))
	}
}

func TestAddressOutputMapper(t *testing.T) {
	output := ec2.DescribeAddressesOutput{
		Addresses: []types.Address{
			{
				PublicIp:           adapterhelpers.PtrString("3.11.82.6"),
				AllocationId:       adapterhelpers.PtrString("eipalloc-030a6f43bc6086267"),
				Domain:             types.DomainTypeVpc,
				PublicIpv4Pool:     adapterhelpers.PtrString("amazon"),
				NetworkBorderGroup: adapterhelpers.PtrString("eu-west-2"),
				InstanceId:         adapterhelpers.PtrString("instance"),
				CarrierIp:          adapterhelpers.PtrString("3.11.82.7"),
				CustomerOwnedIp:    adapterhelpers.PtrString("3.11.82.8"),
				NetworkInterfaceId: adapterhelpers.PtrString("foo"),
				PrivateIpAddress:   adapterhelpers.PtrString("3.11.82.9"),
			},
		},
	}

	items, err := addressOutputMapper(context.Background(), nil, "foo", nil, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].PublicIp,
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].InstanceId,
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].CarrierIp,
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].CustomerOwnedIp,
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-network-interface",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].NetworkInterfaceId,
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  *output.Addresses[0].PrivateIpAddress,
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}

func TestNewEC2AddressAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2AddressAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
