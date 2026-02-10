---
title: GCP Ai Platform Custom Job
sidebar_label: gcp-ai-platform-custom-job
---

A GCP AI Platform Custom Job (now part of Vertex AI) is a fully-managed training workload that runs user-supplied code inside one or more container images on Google Cloud infrastructure. It allows you to specify machine types, accelerators, networking and encryption settings, then orchestrates the provisioning, execution and clean-up of the training cluster. Custom Jobs are typically used when pre-built AutoML options are insufficient and you need complete control over your training loop.  
Official documentation: https://cloud.google.com/vertex-ai/docs/training/create-custom-job

## Supported Methods

- `GET`: Get a gcp-ai-platform-custom-job by its "name"
- `LIST`: List all gcp-ai-platform-custom-job
- ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

A successful Custom Job can optionally upload the trained artefacts as a Vertex AI Model resource; if that happens, the job will reference (and be referenced by) the resulting `gcp-ai-platform-model`.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Custom Jobs support customer-managed encryption keys (CMEK). When a CMEK is specified, the job resource, its logs and any artefacts it creates are encrypted with the referenced `gcp-cloud-kms-crypto-key`.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

You can run Custom Jobs inside a specific VPC network to reach private data sources or to avoid egress to the public internet. In that case the job is linked to the chosen `gcp-compute-network`.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Execution of a Custom Job occurs under a user-specified service account, which determines the permissions the training containers possess. The job therefore has a direct relationship to a `gcp-iam-service-account`.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Training code commonly reads data from, and writes checkpoints or model artefacts to, Cloud Storage. The buckets used for staging, input or output will be surfaced as linked `gcp-storage-bucket` resources.
