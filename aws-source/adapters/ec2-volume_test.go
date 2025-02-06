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

func TestVolumeInputMapperGet(t *testing.T) {
	input, err := volumeInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.VolumeIds) != 1 {
		t.Fatalf("expected 1 Volume ID, got %v", len(input.VolumeIds))
	}

	if input.VolumeIds[0] != "bar" {
		t.Errorf("expected Volume ID to be bar, got %v", input.VolumeIds[0])
	}
}

func TestVolumeInputMapperList(t *testing.T) {
	input, err := volumeInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.VolumeIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestVolumeOutputMapper(t *testing.T) {
	output := &ec2.DescribeVolumesOutput{
		Volumes: []types.Volume{
			{
				Attachments: []types.VolumeAttachment{
					{
						AttachTime:          adapterhelpers.PtrTime(time.Now()),
						Device:              adapterhelpers.PtrString("/dev/sdb"),
						InstanceId:          adapterhelpers.PtrString("i-0667d3ca802741e30"),
						State:               types.VolumeAttachmentStateAttaching,
						VolumeId:            adapterhelpers.PtrString("vol-0eae6976b359d8825"),
						DeleteOnTermination: adapterhelpers.PtrBool(false),
					},
				},
				AvailabilityZone:   adapterhelpers.PtrString("eu-west-2c"),
				CreateTime:         adapterhelpers.PtrTime(time.Now()),
				Encrypted:          adapterhelpers.PtrBool(false),
				Size:               adapterhelpers.PtrInt32(8),
				State:              types.VolumeStateInUse,
				VolumeId:           adapterhelpers.PtrString("vol-0eae6976b359d8825"),
				Iops:               adapterhelpers.PtrInt32(3000),
				VolumeType:         types.VolumeTypeGp3,
				MultiAttachEnabled: adapterhelpers.PtrBool(false),
				Throughput:         adapterhelpers.PtrInt32(125),
			},
		},
	}

	items, err := volumeOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedQuery:  "i-0667d3ca802741e30",
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2VolumeAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2VolumeAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
