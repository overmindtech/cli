---
title: GCP Ai Platform Pipeline Job
sidebar_label: gcp-ai-platform-pipeline-job
---

A GCP AI Platform Pipeline Job (now part of Vertex AI Pipelines) represents a single execution of a machine-learning workflow defined in a Kubeflow/Vertex AI pipeline. The job orchestrates a directed acyclic graph (DAG) of pipeline components such as data preparation, model training and evaluation, and optionally deployment. Each run is stored as a resource that tracks the DAG definition, runtime parameters, execution state, logs and metadata.  
Official documentation: https://cloud.google.com/vertex-ai/docs/pipelines/introduction

## Supported Methods

* `GET`: Get a gcp-ai-platform-pipeline-job by its "name"
* `LIST`: List all gcp-ai-platform-pipeline-job
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the pipeline job is configured to use customer-managed encryption keys (CMEK), the key referenced here encrypts pipeline artefacts such as metadata, intermediate files and model checkpoints.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Pipeline components that run in custom training containers or Dataflow/Dataproc jobs may be attached to a specific VPC network to control egress, ingress and private service access. The pipeline job therefore has an implicit or explicit relationship with the VPC network used at execution time.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The pipeline job executes under a service account which grants it permissions to create and manage downstream resources (e.g. training jobs, storage objects, BigQuery datasets). Overmind links the job to the service account that appears in its runtime configuration.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Vertex AI Pipelines store pipeline definitions, intermediate artefacts, and output models in Cloud Storage. A pipeline job will reference one or more buckets for source code, artefacts and logging, so Overmind creates links to each bucket it touches.
