package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestMountTargetOutputMapper(t *testing.T) {
	output := &efs.DescribeMountTargetsOutput{
		MountTargets: []types.MountTargetDescription{
			{
				FileSystemId:         adapterhelpers.PtrString("fs-1234567890"),
				LifeCycleState:       types.LifeCycleStateAvailable,
				MountTargetId:        adapterhelpers.PtrString("fsmt-01e86506d8165e43f"),
				SubnetId:             adapterhelpers.PtrString("subnet-1234567"),
				AvailabilityZoneId:   adapterhelpers.PtrString("use1-az1"),
				AvailabilityZoneName: adapterhelpers.PtrString("us-east-1"),
				IpAddress:            adapterhelpers.PtrString("10.230.43.1"),
				NetworkInterfaceId:   adapterhelpers.PtrString("eni-2345"),
				OwnerId:              adapterhelpers.PtrString("234234"),
				VpcId:                adapterhelpers.PtrString("vpc-23452345235"),
			},
		},
	}

	items, err := MountTargetOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedType:   "efs-file-system",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "fs-1234567890",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet-1234567",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "10.230.43.1",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "ec2-network-interface",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "eni-2345",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-23452345235",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}
