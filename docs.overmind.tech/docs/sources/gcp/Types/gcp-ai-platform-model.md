---
title: GCP Ai Platform Model
sidebar_label: gcp-ai-platform-model
---

A GCP AI Platform Model (now part of Vertex AI) is a logical container that holds the metadata and artefacts required to serve machine-learning predictions. A model record points to one or more model versions or container images, the Cloud Storage location of the trained parameters, and optional encryption settings. Models are deployed to Endpoints for online prediction or used directly in batch/streaming inference jobs. Official documentation: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models

## Supported Methods

- `GET`: Get a gcp-ai-platform-model by its "name"
- `LIST`: List all gcp-ai-platform-model
- ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-endpoint`](/sources/gcp/Types/gcp-ai-platform-endpoint)

A model is deployed to one or more Endpoints. The link shows where this model is currently serving traffic or could be routed for prediction.

### [`gcp-ai-platform-pipeline-job`](/sources/gcp/Types/gcp-ai-platform-pipeline-job)

Training or transformation Pipeline Jobs often create or update Model resources; linking them highlights which automated workflow produced the model and therefore which code/data lineage applies.

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

If the model is served via a custom prediction container, the Model record references a Docker image stored in Artifact Registry. This link surfaces that underlying image and its associated vulnerabilities.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Models can be protected with customer-managed encryption keys (CMEK). Overmind links the model to the specific KMS key to expose encryption scope and key rotation risks.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

The model’s artefacts (e.g., SavedModel, scikit-learn pickle, PyTorch state) reside in a Cloud Storage bucket referenced by `artifactUri`. Linking to the bucket reveals data-at-rest location and its IAM policy.
