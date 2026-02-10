---
title: GCP Ai Platform Pipeline Job
sidebar_label: gcp-ai-platform-pipeline-job
---

A **GCP AI Platform Pipeline Job** (now part of Vertex AI Pipelines) represents a managed execution of a Kubeflow pipeline on Google Cloud. It orchestrates a series of container-based tasks—such as data preprocessing, model training, and deployment—into a reproducible workflow that runs on Google-managed infrastructure. Each job stores its metadata, intermediate artefacts and logs in Google-hosted services, and can be monitored, retried or version-controlled through the Vertex AI console or API.
For full details, see the official documentation: [Vertex AI Pipelines – Run pipeline jobs](https://docs.cloud.google.com/vertex-ai/docs/pipelines/run-pipeline).

## Supported Methods

- `GET`: Get a gcp-ai-platform-pipeline-job by its "name"
- `LIST`: List all gcp-ai-platform-pipeline-job
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A pipeline job can be configured to use customer-managed encryption keys (CMEK) so that all intermediate artefacts and metadata produced by the pipeline are encrypted with a specific Cloud KMS crypto key. Overmind therefore surfaces a link to the `gcp-cloud-kms-crypto-key` that protects the job’s resources.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Pipeline components often run on GKE clusters or custom training/serving services that are attached to a VPC network. When a job specifies a `network` or `privateClusterConfig`, Overmind links the job to the corresponding `gcp-compute-network`, highlighting network-level exposure or egress restrictions that may affect the pipeline.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every pipeline job executes under a service account whose IAM permissions determine which Google Cloud resources the job can access (e.g. storage buckets, BigQuery datasets). Overmind connects the job to that `gcp-iam-service-account` so that permission scopes and potential privilege escalations can be inspected.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Pipeline jobs read from and write to Cloud Storage for dataset ingestion, model artefact output and pipeline metadata storage. Any bucket referenced in the job’s `pipeline_root`, component arguments or logging configuration is linked here, allowing visibility into data residency, ACLs and lifecycle policies relevant to the pipeline’s operation.
