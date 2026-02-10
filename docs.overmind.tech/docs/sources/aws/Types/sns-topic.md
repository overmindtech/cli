---
title: SNS Topic
sidebar_label: sns-topic
---

An Amazon Simple Notification Service (SNS) topic is a logical access point through which publishers send messages that are then fanned-out to subscribed endpoints such as email addresses, HTTP/S webhooks, Lambda functions or SQS queues. Topics can be configured with attributes such as delivery policies, access control policies and optional server-side encryption using AWS Key Management Service (KMS). For further details refer to the official AWS documentation: https://docs.aws.amazon.com/sns/latest/dg/sns-create-topic.html

**Terrafrom Mappings:**

- `aws_sns_topic.id`

## Supported Methods

- `GET`: Get an SNS topic by its ARN
- `LIST`: List all SNS topics
- `SEARCH`: Search SNS topic by ARN

## Possible Links

### [`kms-key`](/sources/aws/Types/kms-key)

If server-side encryption is enabled for the SNS topic, it references a KMS customer master key (CMK). This link allows Overmind to surface the relationship between the topic and the key that protects its message payloads in transit and at rest.
