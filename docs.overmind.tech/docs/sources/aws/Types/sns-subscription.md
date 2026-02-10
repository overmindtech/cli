---
title: SNS Subscription
sidebar_label: sns-subscription
---

An Amazon Simple Notification Service (SNS) subscription represents the association between an SNS topic and the endpoint that receives the messages published to that topic. Each subscription specifies the delivery protocol (e-mail, SMS, HTTP/S, Lambda, SQS, Firehose, etc.), the endpoint address, and optional delivery policies or filter policies that control how and when messages are delivered. For full details see the official AWS documentation: https://docs.aws.amazon.com/sns/latest/dg/sns-subscription.html

**Terrafrom Mappings:**

- `aws_sns_topic_subscription.id`

## Supported Methods

- `GET`: Get an SNS subscription by its ARN
- `LIST`: List all SNS subscriptions
- `SEARCH`: Search SNS subscription by ARN

## Possible Links

### [`sns-topic`](/sources/aws/Types/sns-topic)

Every subscription belongs to exactly one SNS topic. The subscription’s ARN embeds the topic ARN, and deleting the topic automatically removes the subscription. Overmind links the subscription to its parent `sns-topic` so you can trace message flow from publisher (topic) to consumer (subscription endpoint).

### [`iam-role`](/sources/aws/Types/iam-role)

If the subscription delivers to an AWS resource in another account (e.g., cross-account SQS queue, Lambda function, or Kinesis Data Firehose), SNS must assume an IAM role that grants it permission to publish to that resource. Overmind links the subscription to any `iam-role` referenced in its delivery policy to help you verify that the correct cross-account permissions are in place.
