package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func (t *DynamoDBTestClient) DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error) {
	return &dynamodb.DescribeBackupOutput{
		BackupDescription: &types.BackupDescription{
			BackupDetails: &types.BackupDetails{
				BackupArn:              new("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             new("test2-backup"),
				BackupSizeBytes:        new(int64(0)),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupCreationDateTime: new(time.Now()),
			},
			SourceTableDetails: &types.SourceTableDetails{
				TableName:      new("test2"), // link
				TableId:        new("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:       new("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				TableSizeBytes: new(int64(0)),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: new("ArtistId"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: new("Concert"),
						KeyType:       types.KeyTypeRange,
					},
				},
				TableCreationDateTime: new(time.Now()),
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  new(int64(5)),
					WriteCapacityUnits: new(int64(5)),
				},
				ItemCount:   new(int64(0)),
				BillingMode: types.BillingModeProvisioned,
			},
			SourceTableFeatureDetails: &types.SourceTableFeatureDetails{
				GlobalSecondaryIndexes: []types.GlobalSecondaryIndexInfo{
					{
						IndexName: new("GSI"),
						KeySchema: []types.KeySchemaElement{
							{
								AttributeName: new("TicketSales"),
								KeyType:       types.KeyTypeHash,
							},
						},
						Projection: &types.Projection{
							ProjectionType: types.ProjectionTypeKeysOnly,
						},
						ProvisionedThroughput: &types.ProvisionedThroughput{
							ReadCapacityUnits:  new(int64(5)),
							WriteCapacityUnits: new(int64(5)),
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
				TableName:              new("test2"),
				TableId:                new("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:               new("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				BackupArn:              new("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             new("test2-backup"),
				BackupCreationDateTime: new(time.Now()),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupSizeBytes:        new(int64(10)),
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

	adapter := NewDynamoDBBackupAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
		SkipGet: true,
	}

	test.Run(t)
}
