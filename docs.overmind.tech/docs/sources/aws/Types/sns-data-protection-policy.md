---
title: SNS Data Protection Policy
sidebar_label: sns-data-protection-policy
---

Amazon Simple Notification Service (SNS) is a fully managed messaging service for both application-to-application (A2A) and application-to-person (A2P) communication. SNS topics allow you to fan out messages to a large number of subscribers, including distributed systems, and serverless applications. The SNS Data Protection Policy provides a mechanism to ensure that the data transmitted through SNS is compliant with your organisational and regulatory requirements. This policy is used to define and enforce encryption, data retention, and access control practices on SNS topics. For more details, you can refer to the [official AWS SNS Data Protection documentation](https://docs.aws.amazon.com/sns/latest/dg/sns-data-encryption.html).

**Terraform Mappings:**

- `aws_sns_topic_data_protection_policy.arn`

## Supported Methods

- `GET`: Get an SNS data protection policy by associated topic ARN
- ~~`LIST`~~
- `SEARCH`: Search SNS data protection policies by its ARN

## Possible Links

### [`sns-topic`](/sources/aws/Types/sns-topic)

The SNS Data Protection Policy is directly related to SNS topics as it outlines the security measures and data management practices that are applied to messages sent through these topics. By associating a data protection policy with an SNS topic, users can ensure that their SNS workflows adhere to the necessary data protection and compliance standards.
