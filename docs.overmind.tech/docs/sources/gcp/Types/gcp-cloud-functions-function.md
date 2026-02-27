---
title: GCP Cloud Functions Function
sidebar_label: gcp-cloud-functions-function
---

Google Cloud Functions is a server-less execution environment that lets you run event-driven code without provisioning or managing servers. A “Function” is the deployed piece of code together with its configuration (runtime, memory/CPU limits, environment variables, ingress/egress settings, triggers and IAM bindings). Documentation: https://cloud.google.com/functions/docs

## Supported Methods

* `GET`: Get a gcp-cloud-functions-function by its "locations|functions"
* ~~`LIST`~~
* `SEARCH`: Search for gcp-cloud-functions-function by its "locations"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A function can reference a Cloud KMS crypto key to decrypt secrets or to use Customer-Managed Encryption Keys (CMEK) for its source code stored in Cloud Storage. Overmind therefore links the function to any KMS keys it is authorised to use.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each Cloud Function executes as a service account, and other service accounts may be granted permission to invoke or manage it. Overmind links the function to the runtime service account and to any caller or admin accounts discovered in its IAM policy.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Pub/Sub topics are commonly used as event triggers. When a function is configured to fire on messages published to a topic, Overmind records a link between the function and that topic.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

Second-generation Cloud Functions are built and deployed as Cloud Run services under the hood. Overmind links the function to the underlying Cloud Run service so you can trace configuration and runtime dependencies.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets can be both event sources (object create/delete triggers) and repositories for a function’s source code during deployment. Overmind links the function to any bucket that serves as a trigger or holds its source archive.
