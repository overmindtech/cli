package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestImageInputMapperGet(t *testing.T) {
	input, err := imageInputMapperGet("foo", "az-name")

	if err != nil {
		t.Error(err)
	}

	if len(input.ImageIds) != 1 {
		t.Fatalf("expected 1 zone names, got %v", len(input.ImageIds))
	}

	if input.ImageIds[0] != "az-name" {
		t.Errorf("expected zone name to be to be az-name, got %v", input.ImageIds[0])
	}
}

func TestImageInputMapperList(t *testing.T) {

	input, err := imageInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.ImageIds) != 0 {
		t.Fatalf("expected 0 zone names, got %v", len(input.ImageIds))
	}
}

func TestImageOutputMapper(t *testing.T) {
	output := ec2.DescribeImagesOutput{
		Images: []types.Image{
			{
				Architecture:    "x86_64",
				CreationDate:    adapterhelpers.PtrString("2022-12-16T19:37:36.000Z"),
				ImageId:         adapterhelpers.PtrString("ami-0ed3646be6ecd97c5"),
				ImageLocation:   adapterhelpers.PtrString("052392120703/test"),
				ImageType:       types.ImageTypeValuesMachine,
				Public:          adapterhelpers.PtrBool(false),
				OwnerId:         adapterhelpers.PtrString("052392120703"),
				PlatformDetails: adapterhelpers.PtrString("Linux/UNIX"),
				UsageOperation:  adapterhelpers.PtrString("RunInstances"),
				State:           types.ImageStateAvailable,
				BlockDeviceMappings: []types.BlockDeviceMapping{
					{
						DeviceName: adapterhelpers.PtrString("/dev/xvda"),
						Ebs: &types.EbsBlockDevice{
							DeleteOnTermination: adapterhelpers.PtrBool(true),
							SnapshotId:          adapterhelpers.PtrString("snap-0efd796ecbd599f8d"),
							VolumeSize:          adapterhelpers.PtrInt32(8),
							VolumeType:          types.VolumeTypeGp2,
							Encrypted:           adapterhelpers.PtrBool(false),
						},
					},
				},
				EnaSupport:         adapterhelpers.PtrBool(true),
				Hypervisor:         types.HypervisorTypeXen,
				Name:               adapterhelpers.PtrString("test"),
				RootDeviceName:     adapterhelpers.PtrString("/dev/xvda"),
				RootDeviceType:     types.DeviceTypeEbs,
				SriovNetSupport:    adapterhelpers.PtrString("simple"),
				VirtualizationType: types.VirtualizationTypeHvm,
				Tags: []types.Tag{
					{
						Key:   adapterhelpers.PtrString("Name"),
						Value: adapterhelpers.PtrString("test"),
					},
				},
			},
		},
	}

	items, err := imageOutputMapper(context.Background(), nil, "foo", nil, &output)

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

	if item.UniqueAttributeValue() != *output.Images[0].ImageId {
		t.Errorf("Expected item unique attribute value to be %v, got %v", *output.Images[0].ImageId, item.UniqueAttributeValue())
	}
}

func TestNewEC2ImageAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2ImageAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
