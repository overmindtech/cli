---
title: GCP Cloud Functions Function
sidebar_label: gcp-cloud-functions-function
---

A Google Cloud Functions Function is a serverless, event-driven compute resource that executes user-supplied code in response to HTTP requests or a wide range of Google Cloud events. Because Google Cloud manages the underlying infrastructure, you only specify the code, runtime, memory, timeout, trigger and IAM policy, and you are billed solely for the resources actually consumed while the function is running. For more detail, see Google’s official documentation: https://cloud.google.com/functions/docs/concepts/overview.

## Supported Methods

- `GET`: Get a gcp-cloud-functions-function by its "locations|functions"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-cloud-functions-function by its "locations"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled, the function’s source code, environment variables or secret volumes are encrypted with a Cloud KMS CryptoKey. Overmind links the function to any CryptoKey that protects its assets so you can assess key rotation or deletion risks.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every Cloud Function runs as an IAM Service Account. The permissions granted to this account define what the function can read or modify at runtime. Overmind links the function to its execution service account, allowing you to evaluate privilege levels and potential lateral-movement paths.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

A function can be triggered by a Pub/Sub topic or publish messages to one. Overmind records these relationships so you can see which topics will invoke the function and what downstream systems might be affected if the function misbehaves.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

Second-generation Cloud Functions are deployed on Cloud Run. Overmind links the function to the underlying Cloud Run Service, exposing additional configuration such as VPC connectors, ingress settings and revision history that may introduce risk.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Functions often interact with Cloud Storage: source code may be stored in a staging bucket, and functions can be triggered by bucket events (e.g., object creation). Overmind links the function to any associated buckets, helping you identify data-exfiltration risks and unintended public access.
