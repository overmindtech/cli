package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestAccessPointOutputMapper(t *testing.T) {
	output := &efs.DescribeAccessPointsOutput{
		AccessPoints: []types.AccessPointDescription{
			{
				AccessPointArn: PtrString("arn:aws:elasticfilesystem:eu-west-2:944651592624:access-point/fsap-073b1534eafbc5ee2"),
				AccessPointId:  PtrString("fsap-073b1534eafbc5ee2"),
				ClientToken:    PtrString("pvc-66e4418c-edf5-4a0e-9834-5945598d51fe"),
				FileSystemId:   PtrString("fs-0c6f2f41e957f42a9"),
				LifeCycleState: types.LifeCycleStateAvailable,
				Name:           PtrString("example access point"),
				OwnerId:        PtrString("944651592624"),
				PosixUser: &types.PosixUser{
					Gid: PtrInt64(1000),
					Uid: PtrInt64(1000),
					SecondaryGids: []int64{
						1002,
					},
				},
				RootDirectory: &types.RootDirectory{
					CreationInfo: &types.CreationInfo{
						OwnerGid:    PtrInt64(1000),
						OwnerUid:    PtrInt64(1000),
						Permissions: PtrString("700"),
					},
					Path: PtrString("/etc/foo"),
				},
				Tags: []types.Tag{
					{
						Key:   PtrString("Name"),
						Value: PtrString("example access point"),
					},
				},
			},
		},
	}

	items, err := AccessPointOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedType:   "efs-file-system",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "fs-0c6f2f41e957f42a9",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEFSAccessPointAdapter(t *testing.T) {
	client, account, region := efsGetAutoConfig(t)

	adapter := NewEFSAccessPointAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
