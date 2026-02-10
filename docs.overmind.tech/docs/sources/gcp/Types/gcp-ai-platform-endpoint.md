---
title: GCP Ai Platform Endpoint
sidebar_label: gcp-ai-platform-endpoint
---

A Vertex AI (formerly AI Platform) **Endpoint** is a regional resource that serves as an entry-point for online prediction requests in Google Cloud. One or more trained **Models** can be deployed to an Endpoint, after which client applications invoke the Endpoint’s HTTPS URL (or Private Service Connect address) to obtain real-time predictions. The resource stores configuration such as traffic splitting between models, logging settings, encryption settings and the VPC network to be used for private access.  
Official documentation: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints

## Supported Methods

- `GET`: Get a gcp-ai-platform-endpoint by its "name"
- `LIST`: List all gcp-ai-platform-endpoint
- ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

An Endpoint may contain one or many `deployedModel` blocks, each of which references a separate Model resource. Overmind links the Endpoint to every Model that is currently deployed or that has traffic allocated to it.

### [`gcp-ai-platform-model-deployment-monitoring-job`](/sources/gcp/Types/gcp-ai-platform-model-deployment-monitoring-job)

If model-deployment monitoring has been enabled, the monitoring job resource records statistics and drift detection for a specific Endpoint. Overmind links the Endpoint to all monitoring jobs that target it.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Prediction logging and monitoring can be configured to write request/response data into BigQuery tables. Those tables are therefore linked to the Endpoint that produced the records.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Endpoints can be created with a Customer-Managed Encryption Key (CMEK) via the `encryptionSpec.kmsKeyName` field. Overmind links the Endpoint to the specific Cloud KMS CryptoKey it uses for at-rest encryption.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When an Endpoint is set up for private predictions, it must specify a VPC network (`network` field) that will be used for Private Service Connect. This creates a relationship between the Endpoint and the referenced Compute Network.
