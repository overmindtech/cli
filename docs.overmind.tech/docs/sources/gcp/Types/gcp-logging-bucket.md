---
title: GCP Logging Bucket
sidebar_label: gcp-logging-bucket
---

A GCP Logging Bucket is a regional or multi-regional storage container managed by Cloud Logging that stores log entries routed from one or more Google Cloud projects, folders or organisations. Buckets provide fine-grained control over where logs are kept, how long they are retained, and which encryption keys protect them. Log buckets behave similarly to Cloud Storage buckets, but are optimised for log data and are accessed through the Cloud Logging API rather than through Cloud Storage.  
See the official documentation for full details: https://cloud.google.com/logging/docs/storage

## Supported Methods

- `GET`: Get a gcp-logging-bucket by its "locations|buckets"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-logging-bucket by its "locations"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A logging bucket can be configured to use customer-managed encryption keys (CMEK). When CMEK is enabled, the bucket references a Cloud KMS Crypto Key that holds the symmetric key material used to encrypt and decrypt the stored log entries.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

If CMEK is active, the bucket also keeps track of the specific key version that is currently in use. This link represents the exact Crypto Key Version providing encryption for the bucket at a given point in time.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Cloud Logging uses service accounts to write, read or route logs into a bucket. The bucket’s IAM policy may grant `roles/logging.bucketWriter` or `roles/logging.viewer` to particular service accounts, and the Log Router’s reserved service account must have permission to encrypt data when CMEK is enabled.
