package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestFileSystemOutputMapper(t *testing.T) {
	output := &efs.DescribeFileSystemsOutput{
		FileSystems: []types.FileSystemDescription{
			{
				CreationTime:         PtrTime(time.Now()),
				CreationToken:        PtrString("TOKEN"),
				FileSystemId:         PtrString("fs-1231123123"),
				LifeCycleState:       types.LifeCycleStateAvailable,
				NumberOfMountTargets: 10,
				OwnerId:              PtrString("944651592624"),
				PerformanceMode:      types.PerformanceModeGeneralPurpose,
				SizeInBytes: &types.FileSystemSize{
					Value:           1024,
					Timestamp:       PtrTime(time.Now()),
					ValueInIA:       PtrInt64(2048),
					ValueInStandard: PtrInt64(128),
				},
				Tags: []types.Tag{
					{
						Key:   PtrString("foo"),
						Value: PtrString("bar"),
					},
				},
				AvailabilityZoneId:           PtrString("use1-az1"),
				AvailabilityZoneName:         PtrString("us-east-1"),
				Encrypted:                    PtrBool(true),
				FileSystemArn:                PtrString("arn:aws:elasticfilesystem:eu-west-2:944651592624:file-system/fs-0c6f2f41e957f42a9"),
				KmsKeyId:                     PtrString("arn:aws:kms:eu-west-2:944651592624:key/be76a6fa-d307-41c2-a4e3-cbfba2440747"),
				Name:                         PtrString("test"),
				ProvisionedThroughputInMibps: PtrFloat64(64),
				ThroughputMode:               types.ThroughputModeBursting,
			},
		},
	}

	items, err := FileSystemOutputMapper(context.Background(), nil, "foo", nil, output)

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
			ExpectedType:   "efs-backup-policy",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "fs-1231123123",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:eu-west-2:944651592624:key/be76a6fa-d307-41c2-a4e3-cbfba2440747",
			ExpectedScope:  "944651592624.eu-west-2",
		},
		{
			ExpectedType:   "efs-mount-target",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "fs-1231123123",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

}

func TestNewEFSFileSystemAdapter(t *testing.T) {
	client, account, region := efsGetAutoConfig(t)

	adapter := NewEFSFileSystemAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
