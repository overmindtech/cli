package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *DynamoDBTestClient) DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error) {
	return &dynamodb.DescribeBackupOutput{
		BackupDescription: &types.BackupDescription{
			BackupDetails: &types.BackupDetails{
				BackupArn:              adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             adapterhelpers.PtrString("test2-backup"),
				BackupSizeBytes:        adapterhelpers.PtrInt64(0),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupCreationDateTime: adapterhelpers.PtrTime(time.Now()),
			},
			SourceTableDetails: &types.SourceTableDetails{
				TableName:      adapterhelpers.PtrString("test2"), // link
				TableId:        adapterhelpers.PtrString("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:       adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				TableSizeBytes: adapterhelpers.PtrInt64(0),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: adapterhelpers.PtrString("ArtistId"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: adapterhelpers.PtrString("Concert"),
						KeyType:       types.KeyTypeRange,
					},
				},
				TableCreationDateTime: adapterhelpers.PtrTime(time.Now()),
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  adapterhelpers.PtrInt64(5),
					WriteCapacityUnits: adapterhelpers.PtrInt64(5),
				},
				ItemCount:   adapterhelpers.PtrInt64(0),
				BillingMode: types.BillingModeProvisioned,
			},
			SourceTableFeatureDetails: &types.SourceTableFeatureDetails{
				GlobalSecondaryIndexes: []types.GlobalSecondaryIndexInfo{
					{
						IndexName: adapterhelpers.PtrString("GSI"),
						KeySchema: []types.KeySchemaElement{
							{
								AttributeName: adapterhelpers.PtrString("TicketSales"),
								KeyType:       types.KeyTypeHash,
							},
						},
						Projection: &types.Projection{
							ProjectionType: types.ProjectionTypeKeysOnly,
						},
						ProvisionedThroughput: &types.ProvisionedThroughput{
							ReadCapacityUnits:  adapterhelpers.PtrInt64(5),
							WriteCapacityUnits: adapterhelpers.PtrInt64(5),
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
				TableName:              adapterhelpers.PtrString("test2"),
				TableId:                adapterhelpers.PtrString("12670f3b-8ca1-463b-b15e-f2e27eaf70b0"),
				TableArn:               adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2"),
				BackupArn:              adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test2/backup/01673461724486-a6007753"),
				BackupName:             adapterhelpers.PtrString("test2-backup"),
				BackupCreationDateTime: adapterhelpers.PtrTime(time.Now()),
				BackupStatus:           types.BackupStatusAvailable,
				BackupType:             types.BackupTypeUser,
				BackupSizeBytes:        adapterhelpers.PtrInt64(10),
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

	tests := adapterhelpers.QueryTests{
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
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := dynamodb.NewFromConfig(config)

	adapter := NewDynamoDBBackupAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
		SkipGet: true,
	}

	test.Run(t)
}
