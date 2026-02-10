---
title: Lambda Function
sidebar_label: lambda-function
---

AWS Lambda is a serverless compute service that runs your code in response to events and automatically manages the underlying compute resources for you. A Lambda function is the fundamental execution unit: it contains your application code, runtime settings and configuration such as memory, timeout and environment variables. For a full description see the official AWS documentation: https://docs.aws.amazon.com/lambda/latest/dg/welcome.html

**Terrafrom Mappings:**

- `aws_lambda_function.arn`
- `aws_lambda_function_event_invoke_config.id`
- `aws_lambda_function_url.function_arn`

## Supported Methods

- `GET`: Get a lambda function by name
- `LIST`: List all lambda functions
- `SEARCH`: Search for lambda functions by ARN

## Possible Links

### [`iam-role`](/sources/aws/Types/iam-role)

Each Lambda function is executed with an IAM role (its “execution role”). Overmind links the function to that `iam-role` so you can immediately see what permissions the function has and what downstream resources could be affected by its actions.

### [`s3-bucket`](/sources/aws/Types/s3-bucket)

A Lambda function can be triggered by S3 events (e.g. object creation) or load its deployment artefact from an S3 bucket. Overmind links the function to any referenced `s3-bucket` so you can assess event-driven couplings and code-package storage risks.

### [`sns-topic`](/sources/aws/Types/sns-topic)

Lambda functions may subscribe to, or publish messages to, Amazon SNS topics. When a function is configured as an SNS subscription target, Overmind links it to the relevant `sns-topic` so that you can trace message flows and understand failure blast-radius.

### [`sqs-queue`](/sources/aws/Types/sqs-queue)

Lambda can poll SQS queues as an event source. Overmind establishes a link between the function and the `sqs-queue` it consumes so that queue backlogs, permissions and dead-letter configurations are visible in the dependency graph.

### [`lambda-function`](/sources/aws/Types/lambda-function)

A Lambda function can synchronously or asynchronously invoke another Lambda function (for example, in micro-service fan-out patterns). Overmind links calling and called `lambda-function` resources to expose these internal service dependencies.

### [`elbv2-target-group`](/sources/aws/Types/elbv2-target-group)

Application Load Balancers (ALB) can forward requests to Lambda targets via an ELBv2 target group. Overmind links the function to any associated `elbv2-target-group`, allowing you to see inbound HTTP pathways and evaluate scaling or security implications.
