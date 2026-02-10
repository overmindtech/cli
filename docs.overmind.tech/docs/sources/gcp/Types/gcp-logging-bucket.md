---
title: GCP Logging Bucket
sidebar_label: gcp-logging-bucket
---

A GCP Logging Bucket is a regional or multi-regional storage container within Cloud Logging that holds log entries for long-term retention, analysis and export. Buckets allow you to isolate logs by project, folder or organisation, set individual retention periods, and apply fine-grained IAM policies. They can be configured for customer-managed encryption and for log routing between projects or across the organisation.  
For full details see the Google Cloud documentation: https://cloud.google.com/logging/docs/storage#buckets

## Supported Methods

- `GET`: Get a gcp-logging-bucket by its "locations|buckets"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-logging-bucket by its "locations"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A logging bucket can be encrypted with a customer-managed encryption key (CMEK). When CMEK is enabled, the bucket stores the full resource name of the Cloud KMS crypto key that protects the log data, creating a dependency on that `gcp-cloud-kms-crypto-key` resource.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Writing, reading and routing logs rely on service accounts such as the Log Router and Google-managed writer accounts. These accounts appear in the bucket’s IAM policy and permissions, so the bucket is linked to the corresponding `gcp-iam-service-account` resources.
