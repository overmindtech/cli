package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
						AttachTime:          PtrTime(time.Now()),
						Device:              PtrString("/dev/sdb"),
						InstanceId:          PtrString("i-0667d3ca802741e30"),
						State:               types.VolumeAttachmentStateAttaching,
						VolumeId:            PtrString("vol-0eae6976b359d8825"),
						DeleteOnTermination: PtrBool(false),
					},
				},
				AvailabilityZone:   PtrString("eu-west-2c"),
				CreateTime:         PtrTime(time.Now()),
				Encrypted:          PtrBool(false),
				Size:               PtrInt32(8),
				State:              types.VolumeStateInUse,
				VolumeId:           PtrString("vol-0eae6976b359d8825"),
				Iops:               PtrInt32(3000),
				VolumeType:         types.VolumeTypeGp3,
				MultiAttachEnabled: PtrBool(false),
				Throughput:         PtrInt32(125),
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
	tests := QueryTests{
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

	adapter := NewEC2VolumeAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
