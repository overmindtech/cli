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

func TestNetworkInterfaceInputMapperGet(t *testing.T) {
	input, err := networkInterfaceInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.NetworkInterfaceIds) != 1 {
		t.Fatalf("expected 1 NetworkInterface ID, got %v", len(input.NetworkInterfaceIds))
	}

	if input.NetworkInterfaceIds[0] != "bar" {
		t.Errorf("expected NetworkInterface ID to be bar, got %v", input.NetworkInterfaceIds[0])
	}
}

func TestNetworkInterfaceInputMapperList(t *testing.T) {
	input, err := networkInterfaceInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.NetworkInterfaceIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestNetworkInterfaceInputMapperSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		query        string
		expectFilter bool
		filterName   string
		filterValue  string
		expectENIId  bool
		eniId        string
		expectError  bool
	}{
		{
			name:         "Security group ID",
			query:        "sg-0437857de45b640ce",
			expectFilter: true,
			filterName:   "group-id",
			filterValue:  "sg-0437857de45b640ce",
		},
		{
			name:        "Network interface ARN",
			query:       "arn:aws:ec2:eu-west-2:123456789012:network-interface/eni-0b4652e6f2aa36d78",
			expectENIId: true,
			eniId:       "eni-0b4652e6f2aa36d78",
		},
		{
			name:        "Invalid query",
			query:       "invalid-query",
			expectError: true,
		},
		{
			name:        "Invalid ARN type",
			query:       "arn:aws:ec2:eu-west-2:123456789012:instance/i-1234567890abcdef0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input, err := networkInterfaceInputMapperSearch(context.Background(), nil, "123456789012.eu-west-2", tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for query %s, got nil", tt.query)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for query %s: %v", tt.query, err)
				return
			}

			if tt.expectFilter {
				if len(input.Filters) != 1 {
					t.Errorf("expected 1 filter, got %d", len(input.Filters))
					return
				}
				if *input.Filters[0].Name != tt.filterName {
					t.Errorf("expected filter name %s, got %s", tt.filterName, *input.Filters[0].Name)
				}
				if len(input.Filters[0].Values) != 1 || input.Filters[0].Values[0] != tt.filterValue {
					t.Errorf("expected filter value %s, got %v", tt.filterValue, input.Filters[0].Values)
				}
			}

			if tt.expectENIId {
				if len(input.NetworkInterfaceIds) != 1 {
					t.Errorf("expected 1 network interface ID, got %d", len(input.NetworkInterfaceIds))
					return
				}
				if input.NetworkInterfaceIds[0] != tt.eniId {
					t.Errorf("expected network interface ID %s, got %s", tt.eniId, input.NetworkInterfaceIds[0])
				}
			}
		})
	}
}

func TestNetworkInterfaceOutputMapper(t *testing.T) {
	output := &ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []types.NetworkInterface{
			{
				Association: &types.NetworkInterfaceAssociation{
					AllocationId:  adapterhelpers.PtrString("eipalloc-000a9739291350592"),
					AssociationId: adapterhelpers.PtrString("eipassoc-049cda1f947e5efe6"),
					IpOwnerId:     adapterhelpers.PtrString("052392120703"),
					PublicDnsName: adapterhelpers.PtrString("ec2-18-170-133-9.eu-west-2.compute.amazonaws.com"),
					PublicIp:      adapterhelpers.PtrString("18.170.133.9"),
				},
				Attachment: &types.NetworkInterfaceAttachment{
					AttachmentId:        adapterhelpers.PtrString("ela-attach-03e560efca8c9e5d8"),
					DeleteOnTermination: adapterhelpers.PtrBool(false),
					DeviceIndex:         adapterhelpers.PtrInt32(1),
					InstanceOwnerId:     adapterhelpers.PtrString("amazon-aws"),
					Status:              types.AttachmentStatusAttached,
					InstanceId:          adapterhelpers.PtrString("foo"),
				},
				AvailabilityZone: adapterhelpers.PtrString("eu-west-2b"),
				Description:      adapterhelpers.PtrString("Interface for NAT Gateway nat-0e07f7530ef076766"),
				Groups: []types.GroupIdentifier{
					{
						GroupId:   adapterhelpers.PtrString("group-123"),
						GroupName: adapterhelpers.PtrString("something"),
					},
				},
				InterfaceType: types.NetworkInterfaceTypeNatGateway,
				Ipv6Addresses: []types.NetworkInterfaceIpv6Address{
					{
						Ipv6Address: adapterhelpers.PtrString("2001:db8:1234:0000:0000:0000:0000:0000"),
					},
				},
				MacAddress:         adapterhelpers.PtrString("0a:f4:55:b0:6c:be"),
				NetworkInterfaceId: adapterhelpers.PtrString("eni-0b4652e6f2aa36d78"),
				OwnerId:            adapterhelpers.PtrString("052392120703"),
				PrivateDnsName:     adapterhelpers.PtrString("ip-172-31-35-98.eu-west-2.compute.internal"),
				PrivateIpAddress:   adapterhelpers.PtrString("172.31.35.98"),
				PrivateIpAddresses: []types.NetworkInterfacePrivateIpAddress{
					{
						Association: &types.NetworkInterfaceAssociation{
							AllocationId:    adapterhelpers.PtrString("eipalloc-000a9739291350592"),
							AssociationId:   adapterhelpers.PtrString("eipassoc-049cda1f947e5efe6"),
							IpOwnerId:       adapterhelpers.PtrString("052392120703"),
							PublicDnsName:   adapterhelpers.PtrString("ec2-18-170-133-9.eu-west-2.compute.amazonaws.com"),
							PublicIp:        adapterhelpers.PtrString("18.170.133.9"),
							CarrierIp:       adapterhelpers.PtrString("18.170.133.10"),
							CustomerOwnedIp: adapterhelpers.PtrString("18.170.133.11"),
						},
						Primary:          adapterhelpers.PtrBool(true),
						PrivateDnsName:   adapterhelpers.PtrString("ip-172-31-35-98.eu-west-2.compute.internal"),
						PrivateIpAddress: adapterhelpers.PtrString("172.31.35.98"),
					},
				},
				RequesterId:      adapterhelpers.PtrString("440527171281"),
				RequesterManaged: adapterhelpers.PtrBool(true),
				SourceDestCheck:  adapterhelpers.PtrBool(false),
				Status:           types.NetworkInterfaceStatusInUse,
				SubnetId:         adapterhelpers.PtrString("subnet-0d8ae4b4e07647efa"),
				TagSet:           []types.Tag{},
				VpcId:            adapterhelpers.PtrString("vpc-0d7892e00e573e701"),
			},
		},
	}

	items, err := networkInterfaceOutputMapper(context.Background(), nil, "foo", nil, output)

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

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "foo",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "group-123",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "2001:db8:1234:0000:0000:0000:0000:0000",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ip-172-31-35-98.eu-west-2.compute.internal",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "172.31.35.98",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "18.170.133.9",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "18.170.133.10",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ec2-18-170-133-9.eu-west-2.compute.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "18.170.133.11",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet-0d8ae4b4e07647efa",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2NetworkInterfaceAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2NetworkInterfaceAdapter(client, account, region, nil)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
