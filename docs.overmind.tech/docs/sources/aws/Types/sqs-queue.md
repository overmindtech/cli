---
title: SQS Queue
sidebar_label: sqs-queue
---

Amazon Simple Queue Service (SQS) provides fully-managed message queues that decouple and scale micro-services, distributed systems and serverless applications. A queue acts as a buffer, reliably storing any amount of messages until they are processed and deleted by consumers. Two delivery modes are available – standard (at-least-once, best-effort ordering) and FIFO (exactly-once, ordered). Queues can be encrypted, configured with dead-letter queues, and integrated with other AWS services such as Lambda or SNS.  
For a comprehensive description see the official AWS documentation: https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html

**Terrafrom Mappings:**

- `aws_sqs_queue.id`

## Supported Methods

- `GET`: Get an SQS queue attributes by its URL
- `LIST`: List all SQS queue URLs
- `SEARCH`: Search SQS queue by ARN

## Possible Links

### [`http`](/sources/stdlib/Types/http)

Each SQS queue is identified by an HTTPS URL of the form `https://sqs.<region>.amazonaws.com/<account-id>/<queue-name>`. Overmind represents this URL as an `http` item, so the queue is linked to the corresponding `http` item that models the endpoint used by the AWS API.

### [`lambda-event-source-mapping`](/sources/aws/Types/lambda-event-source-mapping)

When a Lambda function is configured with an event-source mapping that pulls messages from an SQS queue, Overmind creates a `lambda-event-source-mapping` item. The mapping item is linked to the SQS queue it reads from, allowing impact analysis when either the queue or the Lambda configuration changes.
