---
title: Lambda Event Source Mapping
sidebar_label: lambda-event-source-mapping
---

AWS Lambda event source mappings are configuration objects that connect an event-producing resource (for example, an SQS queue, DynamoDB stream, Kinesis data stream or Amazon MQ broker) to a Lambda function. They tell Lambda from which resource to poll, what batch size to use, whether to enable the mapping immediately, and numerous advanced options such as filtering and batching windows. In essence, an event source mapping is the glue that turns an upstream stream or queue into invocations of your function.  
Official documentation: https://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#event-source-mapping

**Terrafrom Mappings:**

- `aws_lambda_event_source_mapping.arn`

## Supported Methods

- `GET`: Get a Lambda event source mapping by UUID
- `LIST`: List all Lambda event source mappings
- `SEARCH`: Search for Lambda event source mappings by Event Source ARN (SQS, DynamoDB, Kinesis, etc.)

## Possible Links

### [`lambda-function`](/sources/aws/Types/lambda-function)

Every event source mapping targets exactly one Lambda function. The mapping’s `FunctionName` points to the ARN of that function, so Overmind will create a link from the mapping to the lambda-function resource it invokes.

### [`dynamodb-table`](/sources/aws/Types/dynamodb-table)

When the event source ARN refers to a DynamoDB stream, the underlying DynamoDB table is important context. Overmind links the mapping to the dynamodb-table that owns the stream so that you can trace how table updates lead to Lambda executions.

### [`sqs-queue`](/sources/aws/Types/sqs-queue)

For SQS, the mapping’s `EventSourceArn` is the ARN of an SQS queue. Linking to the sqs-queue resource lets you understand queue configuration (visibility timeout, encryption, redrive policy) and how it might influence Lambda processing.

### [`rds-db-cluster`](/sources/aws/Types/rds-db-cluster)

If the event source is an Amazon RDS for PostgreSQL or MySQL DB cluster emitting events through Amazon RDS for PostgreSQL logical replication slots (via the `RDS Data API` or Aurora’s `MysqlBinlog` integration), the mapping may reference the cluster’s ARN. Overmind links to the rds-db-cluster so you can assess the impact of database changes on the Lambda workflow.
