---
title: GCP Ai Platform Endpoint
sidebar_label: gcp-ai-platform-endpoint
---

A **Google Cloud AI Platform Endpoint** (now part of Vertex AI) is a regional, fully-managed HTTPS entry point that receives online prediction requests and routes them to one or more deployed models. Endpoints let you perform low-latency, autoscaled inference, apply access controls, add request/response logging and attach monitoring jobs.  
Official documentation: https://cloud.google.com/vertex-ai/docs/predictions/getting-predictions#deploy_model_to_endpoint

## Supported Methods

- `GET`: Get a gcp-ai-platform-endpoint by its "name"
- `LIST`: List all gcp-ai-platform-endpoint
- ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

An Endpoint hosts one or more _DeployedModels_, each of which references a standalone AI Platform/Vertex AI Model resource. The link shows which models are currently deployed to, or have traffic routed through, the endpoint.

### [`gcp-ai-platform-model-deployment-monitoring-job`](/sources/gcp/Types/gcp-ai-platform-model-deployment-monitoring-job)

If data-drift or prediction-quality monitoring has been enabled, a Model Deployment Monitoring Job is attached to the endpoint. This relationship identifies the monitoring configuration that observes traffic on the endpoint.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Prediction request and response payloads can be logged to a BigQuery table when logging is enabled on the endpoint. The link indicates which table is used as the logging sink for the endpoint’s traffic.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Customer-managed encryption keys (CMEK) from Cloud KMS can be specified to encrypt endpoint resources at rest. This link reveals the KMS key protecting the endpoint and its deployed models.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Endpoints can be configured for private service access, allowing prediction traffic to stay within a specified VPC network. The relationship points to the Compute Network that provides the private connectivity for the endpoint.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each deployed model on an endpoint runs under a service account whose permissions govern access to other GCP resources (e.g., storage buckets, KMS keys). The link shows which IAM service account is associated with the endpoint’s runtime.
