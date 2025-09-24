package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type lambdaEventSourceMappingClient interface {
	ListEventSourceMappings(ctx context.Context, params *lambda.ListEventSourceMappingsInput, optFns ...func(*lambda.Options)) (*lambda.ListEventSourceMappingsOutput, error)
	GetEventSourceMapping(ctx context.Context, params *lambda.GetEventSourceMappingInput, optFns ...func(*lambda.Options)) (*lambda.GetEventSourceMappingOutput, error)
}

func eventSourceMappingListFunc(ctx context.Context, client lambdaEventSourceMappingClient, _ string) ([]*types.EventSourceMappingConfiguration, error) {
	out, err := client.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{})
	if err != nil {
		return nil, err
	}

	var items []*types.EventSourceMappingConfiguration
	for _, mapping := range out.EventSourceMappings {
		items = append(items, &mapping)
	}

	return items, nil
}

// convertGetEventSourceMappingOutputToConfiguration converts a GetEventSourceMappingOutput to EventSourceMappingConfiguration
func convertGetEventSourceMappingOutputToConfiguration(output *lambda.GetEventSourceMappingOutput) *types.EventSourceMappingConfiguration {
	return &types.EventSourceMappingConfiguration{
		AmazonManagedKafkaEventSourceConfig: output.AmazonManagedKafkaEventSourceConfig,
		BatchSize:                           output.BatchSize,
		BisectBatchOnFunctionError:          output.BisectBatchOnFunctionError,
		DestinationConfig:                   output.DestinationConfig,
		DocumentDBEventSourceConfig:         output.DocumentDBEventSourceConfig,
		EventSourceArn:                      output.EventSourceArn,
		EventSourceMappingArn:               output.EventSourceMappingArn,
		FilterCriteria:                      output.FilterCriteria,
		FilterCriteriaError:                 output.FilterCriteriaError,
		FunctionArn:                         output.FunctionArn,
		FunctionResponseTypes:               output.FunctionResponseTypes,
		KMSKeyArn:                           output.KMSKeyArn,
		LastModified:                        output.LastModified,
		LastProcessingResult:                output.LastProcessingResult,
		MaximumBatchingWindowInSeconds:      output.MaximumBatchingWindowInSeconds,
		MaximumRecordAgeInSeconds:           output.MaximumRecordAgeInSeconds,
		MaximumRetryAttempts:                output.MaximumRetryAttempts,
		MetricsConfig:                       output.MetricsConfig,
		ParallelizationFactor:               output.ParallelizationFactor,
		ProvisionedPollerConfig:             output.ProvisionedPollerConfig,
		Queues:                              output.Queues,
		ScalingConfig:                       output.ScalingConfig,
		SelfManagedEventSource:              output.SelfManagedEventSource,
		SelfManagedKafkaEventSourceConfig:   output.SelfManagedKafkaEventSourceConfig,
		SourceAccessConfigurations:          output.SourceAccessConfigurations,
		StartingPosition:                    output.StartingPosition,
		StartingPositionTimestamp:           output.StartingPositionTimestamp,
		State:                               output.State,
		StateTransitionReason:               output.StateTransitionReason,
		Topics:                              output.Topics,
		TumblingWindowInSeconds:             output.TumblingWindowInSeconds,
		UUID:                                output.UUID,
	}
}

func eventSourceMappingOutputMapper(query, scope string, awsItem *types.EventSourceMappingConfiguration) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)
	if err != nil {
		return nil, err
	}

	// Set the unique attribute (UUID)
	if awsItem.UUID != nil {
		err = attributes.Set("UUID", *awsItem.UUID)
		if err != nil {
			return nil, err
		}
	}

	item := sdp.Item{
		Type:            "lambda-event-source-mapping",
		UniqueAttribute: "UUID",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to the Lambda function if FunctionArn is present
	if awsItem.FunctionArn != nil {
		parsedARN, err := adapterhelpers.ParseARN(*awsItem.FunctionArn)
		if err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "lambda-function",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *awsItem.FunctionArn,
					Scope:  adapterhelpers.FormatScope(parsedARN.AccountID, parsedARN.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// They are tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	// Link to the event source if EventSourceArn is present
	if awsItem.EventSourceArn != nil {
		parsedARN, err := adapterhelpers.ParseARN(*awsItem.EventSourceArn)
		if err == nil {
			var queryType string

			switch parsedARN.Service {
			case "dynamodb":
				queryType = "dynamodb-table"
			case "kinesis":
				queryType = "kinesis-stream"
			case "sqs":
				queryType = "sqs-queue"
			case "kafka":
				queryType = "kafka-cluster"
			case "mq":
				queryType = "mq-broker"
			// Note: DocumentDB clusters use the RDS service identifier ("rds") in their ARNs.
			// Therefore, we map both RDS and DocumentDB clusters to "rds-db-cluster" here.
			case "rds":
				queryType = "rds-db-cluster"
			default:
				// Skip creating links for unknown services
				queryType = ""
			}

			// Only create link if we have a valid queryType
			if queryType != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   queryType,
						Method: sdp.QueryMethod_SEARCH,
						Query:  *awsItem.EventSourceArn,
						Scope:  adapterhelpers.FormatScope(parsedARN.AccountID, parsedARN.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the event source will affect the mapping
						In: true,
						// Changing the mapping won't affect the event source
						Out: false,
					},
				})
			}
		}
	}

	// Set health status based on state
	if awsItem.State != nil {
		switch *awsItem.State {
		case "Enabled":
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case "Creating":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Deleting":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Disabled":
			item.Health = nil
		case "Enabling":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Updating":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Disabling":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}
	}

	return &item, nil
}

func NewLambdaEventSourceMappingAdapter(client lambdaEventSourceMappingClient, accountID string, region string) *adapterhelpers.GetListAdapter[*types.EventSourceMappingConfiguration, lambdaEventSourceMappingClient, *lambda.Options] {
	return &adapterhelpers.GetListAdapter[*types.EventSourceMappingConfiguration, lambdaEventSourceMappingClient, *lambda.Options]{
		ItemType:        "lambda-event-source-mapping",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: lambdaEventSourceMappingAdapterMetadata,
		GetFunc: func(ctx context.Context, client lambdaEventSourceMappingClient, scope, query string) (*types.EventSourceMappingConfiguration, error) {
			out, err := client.GetEventSourceMapping(ctx, &lambda.GetEventSourceMappingInput{
				UUID: &query,
			})
			if err != nil {
				return nil, err
			}
			return convertGetEventSourceMappingOutputToConfiguration(out), nil
		},
		ListFunc: eventSourceMappingListFunc,
		SearchFunc: func(ctx context.Context, client lambdaEventSourceMappingClient, scope string, query string) ([]*types.EventSourceMappingConfiguration, error) {
			// Use the query directly as event source ARN input to ListEventSourceMappings
			out, err := client.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
				EventSourceArn: &query,
			})
			if err != nil {
				return nil, err
			}

			response := make([]*types.EventSourceMappingConfiguration, 0, len(out.EventSourceMappings))
			for _, mapping := range out.EventSourceMappings {
				response = append(response, &mapping)
			}

			return response, nil
		},
		ItemMapper: func(query, scope string, awsItem *types.EventSourceMappingConfiguration) (*sdp.Item, error) {
			return eventSourceMappingOutputMapper(query, scope, awsItem)
		},
	}
}

var lambdaEventSourceMappingAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "lambda-event-source-mapping",
	DescriptiveName: "Lambda Event Source Mapping",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		List:              true,
		GetDescription:    "Get a Lambda event source mapping by UUID",
		SearchDescription: "Search for Lambda event source mappings by Event Source ARN (SQS, DynamoDB, Kinesis, etc.)",
		ListDescription:   "List all Lambda event source mappings",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_lambda_event_source_mapping.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{
		"lambda-function",
		"dynamodb-table",
		"kinesis-stream",
		"sqs-queue",
		"kafka-cluster",
		"mq-broker",
		"rds-db-cluster",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
