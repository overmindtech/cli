package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestFileSystemOutputMapper(t *testing.T) {
	output := &efs.DescribeFileSystemsOutput{
		FileSystems: []types.FileSystemDescription{
			{
				CreationTime:         adapterhelpers.PtrTime(time.Now()),
				CreationToken:        adapterhelpers.PtrString("TOKEN"),
				FileSystemId:         adapterhelpers.PtrString("fs-1231123123"),
				LifeCycleState:       types.LifeCycleStateAvailable,
				NumberOfMountTargets: 10,
				OwnerId:              adapterhelpers.PtrString("944651592624"),
				PerformanceMode:      types.PerformanceModeGeneralPurpose,
				SizeInBytes: &types.FileSystemSize{
					Value:           1024,
					Timestamp:       adapterhelpers.PtrTime(time.Now()),
					ValueInIA:       adapterhelpers.PtrInt64(2048),
					ValueInStandard: adapterhelpers.PtrInt64(128),
				},
				Tags: []types.Tag{
					{
						Key:   adapterhelpers.PtrString("foo"),
						Value: adapterhelpers.PtrString("bar"),
					},
				},
				AvailabilityZoneId:           adapterhelpers.PtrString("use1-az1"),
				AvailabilityZoneName:         adapterhelpers.PtrString("us-east-1"),
				Encrypted:                    adapterhelpers.PtrBool(true),
				FileSystemArn:                adapterhelpers.PtrString("arn:aws:elasticfilesystem:eu-west-2:944651592624:file-system/fs-0c6f2f41e957f42a9"),
				KmsKeyId:                     adapterhelpers.PtrString("arn:aws:kms:eu-west-2:944651592624:key/be76a6fa-d307-41c2-a4e3-cbfba2440747"),
				Name:                         adapterhelpers.PtrString("test"),
				ProvisionedThroughputInMibps: adapterhelpers.PtrFloat64(64),
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
	tests := adapterhelpers.QueryTests{
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

	adapter := NewEFSFileSystemAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
