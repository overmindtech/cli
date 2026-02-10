---
title: GCP Ai Platform Model
sidebar_label: gcp-ai-platform-model
---

A **GCP AI Platform Model** (now part of Vertex AI) is a top-level resource that represents a machine-learning model and its metadata. It groups together one or more model versions (or “Model resources” in Vertex AI terminology), defines the serving container, encryption settings and access controls, and can be deployed to online prediction endpoints or used by batch prediction jobs.  
For full details, see the official documentation: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models

## Supported Methods

- `GET`: Get a gcp-ai-platform-model by its "name"
- `LIST`: List all gcp-ai-platform-model
- ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-endpoint`](/sources/gcp/Types/gcp-ai-platform-endpoint)

An AI Platform Model can be deployed to one or more endpoints. When Overmind detects that a model has been deployed, it links the model to the corresponding `gcp-ai-platform-endpoint` resource so that you can see where the model is serving traffic.

### [`gcp-ai-platform-pipeline-job`](/sources/gcp/Types/gcp-ai-platform-pipeline-job)

Vertex AI Pipeline Jobs often produce models as artefacts at the end of a training pipeline. Overmind links a `gcp-ai-platform-pipeline-job` to the `gcp-ai-platform-model` it created (or updated) so you can trace the provenance of a model back to the pipeline run that generated it.

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

Models use a container image for prediction service. If that container image is stored in Artifact Registry, Overmind establishes a link between the model and the `gcp-artifact-registry-docker-image` representing the serving container. This highlights dependencies on specific container images and versions.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled for the model, the model resource references the Cloud KMS Crypto Key used to encrypt the model data at rest. Overmind links the model to the `gcp-cloud-kms-crypto-key` to surface encryption dependencies and potential key-rotation risks.
