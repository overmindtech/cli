package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
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
				CreationDate:    new("2022-12-16T19:37:36.000Z"),
				ImageId:         new("ami-0ed3646be6ecd97c5"),
				ImageLocation:   new("052392120703/test"),
				ImageType:       types.ImageTypeValuesMachine,
				Public:          new(false),
				OwnerId:         new("052392120703"),
				PlatformDetails: new("Linux/UNIX"),
				UsageOperation:  new("RunInstances"),
				State:           types.ImageStateAvailable,
				BlockDeviceMappings: []types.BlockDeviceMapping{
					{
						DeviceName: new("/dev/xvda"),
						Ebs: &types.EbsBlockDevice{
							DeleteOnTermination: new(true),
							SnapshotId:          new("snap-0efd796ecbd599f8d"),
							VolumeSize:          new(int32(8)),
							VolumeType:          types.VolumeTypeGp2,
							Encrypted:           new(false),
						},
					},
				},
				EnaSupport:         new(true),
				Hypervisor:         types.HypervisorTypeXen,
				Name:               new("test"),
				RootDeviceName:     new("/dev/xvda"),
				RootDeviceType:     types.DeviceTypeEbs,
				SriovNetSupport:    new("simple"),
				VirtualizationType: types.VirtualizationTypeHvm,
				Tags: []types.Tag{
					{
						Key:   new("Name"),
						Value: new("test"),
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

	adapter := NewEC2ImageAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
