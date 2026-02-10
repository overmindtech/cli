---
title: S3 Bucket
sidebar_label: s3-bucket
---

Amazon S3 (Simple Storage Service) buckets are globally-unique containers used to store and organise objects such as files, logs and backups. Each bucket is created within a specific AWS Region, can be configured with fine-grained access controls, lifecycle rules, encryption, versioning and event notifications, and can serve as the origin for many other AWS services. Full service documentation is available in the AWS User Guide: https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html

**Terrafrom Mappings:**

- `aws_s3_bucket_acl.bucket`
- `aws_s3_bucket_analytics_configuration.bucket`
- `aws_s3_bucket_cors_configuration.bucket`
- `aws_s3_bucket_intelligent_tiering_configuration.bucket`
- `aws_s3_bucket_inventory.bucket`
- `aws_s3_bucket_lifecycle_configuration.bucket`
- `aws_s3_bucket_logging.bucket`
- `aws_s3_bucket_metric.bucket`
- `aws_s3_bucket_notification.bucket`
- `aws_s3_bucket_object_lock_configuration.bucket`
- `aws_s3_bucket_object.bucket`
- `aws_s3_bucket_ownership_controls.bucket`
- `aws_s3_bucket_policy.bucket`
- `aws_s3_bucket_public_access_block.bucket`
- `aws_s3_bucket_replication_configuration.bucket`
- `aws_s3_bucket_request_payment_configuration.bucket`
- `aws_s3_bucket_server_side_encryption_configuration.bucket`
- `aws_s3_bucket_versioning.bucket`
- `aws_s3_bucket_website_configuration.bucket`
- `aws_s3_bucket.id`
- `aws_s3_object_copy.bucket`
- `aws_s3_object.bucket`

## Supported Methods

- `GET`: Get an S3 bucket by name
- `LIST`: List all S3 buckets
- `SEARCH`: Search for S3 buckets by ARN

## Possible Links

### [`lambda-function`](/sources/aws/Types/lambda-function)

An S3 bucket can invoke Lambda functions through S3 event notifications (e.g. when an object is created, deleted or restored). Overmind surfaces this relationship so that you can identify deployment risks such as circular triggers or permissions gaps between the bucket and the associated Lambda execution role.

### [`sqs-queue`](/sources/aws/Types/sqs-queue)

Buckets may be configured to send event notifications to SQS queues. Overmind links the bucket to any target queue, allowing you to assess the impact of queue deletion, encryption settings or IAM policies on the integrity of the event pipeline.

### [`sns-topic`](/sources/aws/Types/sns-topic)

Similar to SQS, S3 buckets can publish object-level events to SNS topics. Overmind records this connection so you can verify that topic policies permit delivery and that message fan-out will still function after your planned changes.

### [`s3-bucket`](/sources/aws/Types/s3-bucket)

Buckets are often paired through cross-Region replication or configured as website redirects to one another. Overmind creates links between the source and destination buckets to highlight dependencies such as replication roles, encryption configuration compatibility and versioning status.
