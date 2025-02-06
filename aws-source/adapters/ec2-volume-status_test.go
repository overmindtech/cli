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

func TestVolumeStatusInputMapperGet(t *testing.T) {
	input, err := volumeStatusInputMapperGet("foo", "bar")

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

func TestVolumeStatusInputMapperList(t *testing.T) {
	input, err := volumeStatusInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.VolumeIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestVolumeStatusOutputMapper(t *testing.T) {
	output := &ec2.DescribeVolumeStatusOutput{
		VolumeStatuses: []types.VolumeStatusItem{
			{
				Actions: []types.VolumeStatusAction{
					{
						Code:        adapterhelpers.PtrString("enable-volume-io"),
						Description: adapterhelpers.PtrString("Enable volume I/O"),
						EventId:     adapterhelpers.PtrString("12"),
						EventType:   adapterhelpers.PtrString("io-enabled"),
					},
				},
				AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
				Events: []types.VolumeStatusEvent{
					{
						Description: adapterhelpers.PtrString("The volume is operating normally"),
						EventId:     adapterhelpers.PtrString("12"),
						EventType:   adapterhelpers.PtrString("io-enabled"),
						InstanceId:  adapterhelpers.PtrString("i-0667d3ca802741e30"), // link
						NotAfter:    adapterhelpers.PtrTime(time.Now()),
						NotBefore:   adapterhelpers.PtrTime(time.Now()),
					},
				},
				VolumeId: adapterhelpers.PtrString("vol-0a38796ac85e21c11"), // link
				VolumeStatus: &types.VolumeStatusInfo{
					Details: []types.VolumeStatusDetails{
						{
							Name:   types.VolumeStatusNameIoEnabled,
							Status: adapterhelpers.PtrString("passed"),
						},
						{
							Name:   types.VolumeStatusNameIoPerformance,
							Status: adapterhelpers.PtrString("not-applicable"),
						},
					},
					Status: types.VolumeStatusInfoStatusOk,
				},
			},
		},
	}

	items, err := volumeStatusOutputMapper(context.Background(), nil, "foo", nil, output)

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
		{
			ExpectedType:   "ec2-volume",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vol-0a38796ac85e21c11",
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)
}

func TestNewEC2VolumeStatusAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2VolumeAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
