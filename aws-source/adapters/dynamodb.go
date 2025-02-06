package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Client interface {
	DescribeKinesisStreamingDestination(ctx context.Context, params *dynamodb.DescribeKinesisStreamingDestinationInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeKinesisStreamingDestinationOutput, error)
	DescribeBackup(ctx context.Context, params *dynamodb.DescribeBackupInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeBackupOutput, error)
	ListBackups(ctx context.Context, params *dynamodb.ListBackupsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListBackupsOutput, error)
	ListTagsOfResource(ctx context.Context, params *dynamodb.ListTagsOfResourceInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTagsOfResourceOutput, error)

	dynamodb.DescribeTableAPIClient
	dynamodb.ListTablesAPIClient
}
