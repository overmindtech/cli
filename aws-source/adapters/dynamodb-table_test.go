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

func (t *DynamoDBTestClient) DescribeTable(context.Context, *dynamodb.DescribeTableInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	return &dynamodb.DescribeTableOutput{
		Table: &types.TableDescription{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: new("ArtistId"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: new("Concert"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: new("TicketSales"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			TableName: new("test-DDBTable-1X52D7BWAAB2H"),
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
			TableStatus:      types.TableStatusActive,
			CreationDateTime: new(time.Now()),
			ProvisionedThroughput: &types.ProvisionedThroughputDescription{
				NumberOfDecreasesToday: new(int64(0)),
				ReadCapacityUnits:      new(int64(5)),
				WriteCapacityUnits:     new(int64(5)),
			},
			TableSizeBytes: new(int64(0)),
			ItemCount:      new(int64(0)),
			TableArn:       new("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H"),
			TableId:        new("32ef65bf-d6f3-4508-a3db-f201df09e437"),
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndexDescription{
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
					IndexStatus: types.IndexStatusActive,
					ProvisionedThroughput: &types.ProvisionedThroughputDescription{
						NumberOfDecreasesToday: new(int64(0)),
						ReadCapacityUnits:      new(int64(5)),
						WriteCapacityUnits:     new(int64(5)),
					},
					IndexSizeBytes: new(int64(0)),
					ItemCount:      new(int64(0)),
					IndexArn:       new("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/index/GSI"), // no link, t
				},
			},
			ArchivalSummary: &types.ArchivalSummary{
				ArchivalBackupArn: new("arn:aws:backups:eu-west-1:052392120703:some-backup/one"), // link
				ArchivalDateTime:  new(time.Now()),
				ArchivalReason:    new("fear"),
			},
			BillingModeSummary: &types.BillingModeSummary{
				BillingMode: types.BillingModePayPerRequest,
			},
			GlobalTableVersion: new("1"),
			LatestStreamArn:    new("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/stream/2023-01-11T16:53:02.371"), // This doesn't get linked because there is no more data to get
			LatestStreamLabel:  new("2023-01-11T16:53:02.371"),
			LocalSecondaryIndexes: []types.LocalSecondaryIndexDescription{
				{
					IndexArn:       new("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/index/GSX"), // no link
					IndexName:      new("GSX"),
					IndexSizeBytes: new(int64(29103)),
					ItemCount:      new(int64(234234)),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: new("TicketSales"),
							KeyType:       types.KeyTypeHash,
						},
					},
					Projection: &types.Projection{
						NonKeyAttributes: []string{
							"att1",
						},
						ProjectionType: types.ProjectionTypeInclude,
					},
				},
			},
			Replicas: []types.ReplicaDescription{
				{
					GlobalSecondaryIndexes: []types.ReplicaGlobalSecondaryIndexDescription{
						{
							IndexName: new("name"),
						},
					},
					KMSMasterKeyId: new("keyID"),
					RegionName:     new("eu-west-2"), // link
					ReplicaStatus:  types.ReplicaStatusActive,
					ReplicaTableClassSummary: &types.TableClassSummary{
						TableClass: types.TableClassStandard,
					},
				},
			},
			RestoreSummary: &types.RestoreSummary{
				RestoreDateTime:   new(time.Now()),
				RestoreInProgress: new(false),
				SourceBackupArn:   new("arn:aws:backup:eu-west-1:052392120703:recovery-point:89d0f956-d3a6-42fd-abbd-7d397766bc7e"), // link
				SourceTableArn:    new("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H"),                 // link
			},
			SSEDescription: &types.SSEDescription{
				InaccessibleEncryptionDateTime: new(time.Now()),
				KMSMasterKeyArn:                new("arn:aws:service:region:account:type/id"), // link
				SSEType:                        types.SSETypeAes256,
				Status:                         types.SSEStatusDisabling,
			},
			StreamSpecification: &types.StreamSpecification{
				StreamEnabled:  new(true),
				StreamViewType: types.StreamViewTypeKeysOnly,
			},
			TableClassSummary: &types.TableClassSummary{
				LastUpdateDateTime: new(time.Now()),
				TableClass:         types.TableClassStandard,
			},
		},
	}, nil
}

func (t *DynamoDBTestClient) ListTables(context.Context, *dynamodb.ListTablesInput, ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &dynamodb.ListTablesOutput{
		TableNames: []string{
			"test-DDBTable-1X52D7BWAAB2H",
		},
	}, nil
}

func (t *DynamoDBTestClient) DescribeKinesisStreamingDestination(ctx context.Context, params *dynamodb.DescribeKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeKinesisStreamingDestinationOutput, error) {
	return &dynamodb.DescribeKinesisStreamingDestinationOutput{
		KinesisDataStreamDestinations: []types.KinesisDataStreamDestination{
			{
				DestinationStatus:            types.DestinationStatusActive,
				DestinationStatusDescription: new("description"),
				StreamArn:                    new("arn:aws:kinesis:eu-west-1:052392120703:stream/test"),
			},
		},
	}, nil
}

func (t *DynamoDBTestClient) ListTagsOfResource(context.Context, *dynamodb.ListTagsOfResourceInput, ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error) {
	return &dynamodb.ListTagsOfResourceOutput{
		Tags: []types.Tag{
			{
				Key:   new("key"),
				Value: new("value"),
			},
		},
		NextToken: nil,
	}, nil
}

func TestTableGetFunc(t *testing.T) {
	item, err := tableGetFunc(context.Background(), &DynamoDBTestClient{}, "foo", &dynamodb.DescribeTableInput{})

	if err != nil {
		t.Fatal(err)
	}

	if item.GetTags()["key"] != "value" {
		t.Errorf("expected tag key to be 'value', got '%s'", item.GetTags()["key"])
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "kinesis-stream",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kinesis:eu-west-1:052392120703:stream/test",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "backup-recovery-point",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:backup:eu-west-1:052392120703:recovery-point:89d0f956-d3a6-42fd-abbd-7d397766bc7e",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "dynamodb-table",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
	}

	tests.Execute(t, item)
}

func TestNewDynamoDBTableAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := dynamodb.NewFromConfig(config)

	adapter := NewDynamoDBTableAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
