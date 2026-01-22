package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *DynamoDBTestClient) DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error) {
	return &dynamodb.DescribeBackupOutput{
		BackupDescription: &types.BackupDescription{
			BackupDetails: &types.BackupDetails{
				BackupArn:              PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             PtrString("test2-backup"),
				BackupSizeBytes:        PtrInt64(0),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupCreationDateTime: PtrTime(time.Now()),
			},
			SourceTableDetails: &types.SourceTableDetails{
				TableName:      PtrString("test2"), // link
				TableId:        PtrString("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:       PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				TableSizeBytes: PtrInt64(0),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: PtrString("ArtistId"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: PtrString("Concert"),
						KeyType:       types.KeyTypeRange,
					},
				},
				TableCreationDateTime: PtrTime(time.Now()),
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  PtrInt64(5),
					WriteCapacityUnits: PtrInt64(5),
				},
				ItemCount:   PtrInt64(0),
				BillingMode: types.BillingModeProvisioned,
			},
			SourceTableFeatureDetails: &types.SourceTableFeatureDetails{
				GlobalSecondaryIndexes: []types.GlobalSecondaryIndexInfo{
					{
						IndexName: PtrString("GSI"),
						KeySchema: []types.KeySchemaElement{
							{
								AttributeName: PtrString("TicketSales"),
								KeyType:       types.KeyTypeHash,
							},
						},
						Projection: &types.Projection{
							ProjectionType: types.ProjectionTypeKeysOnly,
						},
						ProvisionedThroughput: &types.ProvisionedThroughput{
							ReadCapacityUnits:  PtrInt64(5),
							WriteCapacityUnits: PtrInt64(5),
						},
					},
				},
			},
		},
	}, nil
}

func (t *DynamoDBTestClient) ListBackups(ctx context.Context, params *dynamodb.ListBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListBackupsOutput, error) {
	return &dynamodb.ListBackupsOutput{
		BackupSummaries: []types.BackupSummary{
			{
				TableName:              PtrString("test2"),
				TableId:                PtrString("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:               PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				BackupArn:              PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             PtrString("test2-backup"),
				BackupCreationDateTime: PtrTime(time.Now()),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupSizeBytes:        PtrInt64(10),
			},
		},
	}, nil
}

func TestBackupGetFunc(t *testing.T) {
	item, err := backupGetFunc(context.Background(), &DynamoDBTestClient{}, "foo", &dynamodb.DescribeBackupInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "dynamodb-table",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "test2",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewDynamoDBBackupAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := dynamodb.NewFromConfig(config)

	adapter := NewDynamoDBBackupAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
		SkipGet: true,
	}

	test.Run(t)
}
