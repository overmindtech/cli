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

func (t *DynamoDBTestClient) DescribeTable(context.Context, *dynamodb.DescribeTableInput, ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	return &dynamodb.DescribeTableOutput{
		Table: &types.TableDescription{
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: adapterhelpers.PtrString("ArtistId"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: adapterhelpers.PtrString("Concert"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: adapterhelpers.PtrString("TicketSales"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			TableName: adapterhelpers.PtrString("test-DDBTable-1X52D7BWAAB2H"),
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
			TableStatus:      types.TableStatusActive,
			CreationDateTime: adapterhelpers.PtrTime(time.Now()),
			ProvisionedThroughput: &types.ProvisionedThroughputDescription{
				NumberOfDecreasesToday: adapterhelpers.PtrInt64(0),
				ReadCapacityUnits:      adapterhelpers.PtrInt64(5),
				WriteCapacityUnits:     adapterhelpers.PtrInt64(5),
			},
			TableSizeBytes: adapterhelpers.PtrInt64(0),
			ItemCount:      adapterhelpers.PtrInt64(0),
			TableArn:       adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H"),
			TableId:        adapterhelpers.PtrString("32ef65bf-d6f3-4508-a3db-f201df09e437"),
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndexDescription{
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
					IndexStatus: types.IndexStatusActive,
					ProvisionedThroughput: &types.ProvisionedThroughputDescription{
						NumberOfDecreasesToday: adapterhelpers.PtrInt64(0),
						ReadCapacityUnits:      adapterhelpers.PtrInt64(5),
						WriteCapacityUnits:     adapterhelpers.PtrInt64(5),
					},
					IndexSizeBytes: adapterhelpers.PtrInt64(0),
					ItemCount:      adapterhelpers.PtrInt64(0),
					IndexArn:       adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/index/GSI"), // no link, t
				},
			},
			ArchivalSummary: &types.ArchivalSummary{
				ArchivalBackupArn: adapterhelpers.PtrString("arn:aws:backups:eu-west-1:052392120703:some-backup/one"), // link
				ArchivalDateTime:  adapterhelpers.PtrTime(time.Now()),
				ArchivalReason:    adapterhelpers.PtrString("fear"),
			},
			BillingModeSummary: &types.BillingModeSummary{
				BillingMode: types.BillingModePayPerRequest,
			},
			GlobalTableVersion: adapterhelpers.PtrString("1"),
			LatestStreamArn:    adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/stream/2023-01-11T16:53:02.371"), // This doesn't get linked because there is no more data to get
			LatestStreamLabel:  adapterhelpers.PtrString("2023-01-11T16:53:02.371"),
			LocalSecondaryIndexes: []types.LocalSecondaryIndexDescription{
				{
					IndexArn:       adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H/index/GSX"), // no link
					IndexName:      adapterhelpers.PtrString("GSX"),
					IndexSizeBytes: adapterhelpers.PtrInt64(29103),
					ItemCount:      adapterhelpers.PtrInt64(234234),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: adapterhelpers.PtrString("TicketSales"),
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
							IndexName: adapterhelpers.PtrString("name"),
						},
					},
					KMSMasterKeyId: adapterhelpers.PtrString("keyID"),
					RegionName:     adapterhelpers.PtrString("eu-west-2"), // link
					ReplicaStatus:  types.ReplicaStatusActive,
					ReplicaTableClassSummary: &types.TableClassSummary{
						TableClass: types.TableClassStandard,
					},
				},
			},
			RestoreSummary: &types.RestoreSummary{
				RestoreDateTime:   adapterhelpers.PtrTime(time.Now()),
				RestoreInProgress: adapterhelpers.PtrBool(false),
				SourceBackupArn:   adapterhelpers.PtrString("arn:aws:backup:eu-west-1:052392120703:recovery-point:89d0f956-d3a6-42fd-abbd-7d397766bc7e"), // link
				SourceTableArn:    adapterhelpers.PtrString("arn:aws:dynamodb:eu-west-1:052392120703:table/test-DDBTable-1X52D7BWAAB2H"),                 // link
			},
			SSEDescription: &types.SSEDescription{
				InaccessibleEncryptionDateTime: adapterhelpers.PtrTime(time.Now()),
				KMSMasterKeyArn:                adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
				SSEType:                        types.SSETypeAes256,
				Status:                         types.SSEStatusDisabling,
			},
			StreamSpecification: &types.StreamSpecification{
				StreamEnabled:  adapterhelpers.PtrBool(true),
				StreamViewType: types.StreamViewTypeKeysOnly,
			},
			TableClassSummary: &types.TableClassSummary{
				LastUpdateDateTime: adapterhelpers.PtrTime(time.Now()),
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
				DestinationStatusDescription: adapterhelpers.PtrString("description"),
				StreamArn:                    adapterhelpers.PtrString("arn:aws:kinesis:eu-west-1:052392120703:stream/test"),
			},
		},
	}, nil
}

func (t *DynamoDBTestClient) ListTagsOfResource(context.Context, *dynamodb.ListTagsOfResourceInput, ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error) {
	return &dynamodb.ListTagsOfResourceOutput{
		Tags: []types.Tag{
			{
				Key:   adapterhelpers.PtrString("key"),
				Value: adapterhelpers.PtrString("value"),
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

	tests := adapterhelpers.QueryTests{
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
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := dynamodb.NewFromConfig(config)

	adapter := NewDynamoDBTableAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
