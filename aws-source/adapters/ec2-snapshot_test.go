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

func TestSnapshotInputMapperGet(t *testing.T) {
	input, err := snapshotInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.SnapshotIds) != 1 {
		t.Fatalf("expected 1 Snapshot ID, got %v", len(input.SnapshotIds))
	}

	if input.SnapshotIds[0] != "bar" {
		t.Errorf("expected Snapshot ID to be bar, got %v", input.SnapshotIds[0])
	}
}

func TestSnapshotInputMapperList(t *testing.T) {
	input, err := snapshotInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.SnapshotIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestSnapshotOutputMapper(t *testing.T) {
	output := &ec2.DescribeSnapshotsOutput{
		Snapshots: []types.Snapshot{
			{
				DataEncryptionKeyId: adapterhelpers.PtrString("ek"),
				KmsKeyId:            adapterhelpers.PtrString("key"),
				SnapshotId:          adapterhelpers.PtrString("id"),
				Description:         adapterhelpers.PtrString("foo"),
				Encrypted:           adapterhelpers.PtrBool(false),
				OutpostArn:          adapterhelpers.PtrString("something"),
				OwnerAlias:          adapterhelpers.PtrString("something"),
				OwnerId:             adapterhelpers.PtrString("owner"),
				Progress:            adapterhelpers.PtrString("50%"),
				RestoreExpiryTime:   adapterhelpers.PtrTime(time.Now()),
				StartTime:           adapterhelpers.PtrTime(time.Now()),
				State:               types.SnapshotStatePending,
				StateMessage:        adapterhelpers.PtrString("pending"),
				StorageTier:         types.StorageTierArchive,
				Tags:                []types.Tag{},
				VolumeId:            adapterhelpers.PtrString("volumeId"),
				VolumeSize:          adapterhelpers.PtrInt32(1024),
			},
		},
	}

	items, err := snapshotOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-volume",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "volumeId",
			ExpectedScope:  item.GetScope(),
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2SnapshotAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2SnapshotAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
