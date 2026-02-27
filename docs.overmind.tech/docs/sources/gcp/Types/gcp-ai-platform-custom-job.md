---
title: GCP Ai Platform Custom Job
sidebar_label: gcp-ai-platform-custom-job
---

A Vertex AI / AI Platform Custom Job represents an ad-hoc machine-learning workload that you want Google Cloud to run on managed infrastructure. By pointing the job at a custom container image or a Python package, you can execute training, hyper-parameter tuning or batch-processing logic with fine-grained control over machine types, accelerators, networking and encryption. The job definition is submitted to the `projects.locations.customJobs` API and Google Cloud provisions the required compute, streams logs, stores artefacts and tears the resources down once the job finishes.  
Official documentation: https://cloud.google.com/vertex-ai/docs/training/create-custom-job

## Supported Methods

* `GET`: Get a gcp-ai-platform-custom-job by its "name"
* `LIST`: List all gcp-ai-platform-custom-job
* ~~`SEARCH`~~

## Possible Links

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

A successful Custom Job can optionally call `model.upload()` or configure `model_to_upload`, causing Vertex AI to register a `Model` resource containing the trained artefacts. Overmind links the job to the resulting `gcp-ai-platform-model` so you can trace how the model was produced.

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

Custom Jobs usually run inside user-supplied container images. When the image is stored in Artifact Registry, Overmind records a link between the job and the specific `gcp-artifact-registry-docker-image` it pulled, making it easy to audit code and dependency provenance.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If you enable customer-managed encryption keys (CMEK) for the job, Google Cloud encrypts logs, checkpoints and model files with the specified KMS key. The job therefore references a `gcp-cloud-kms-crypto-key`, which Overmind surfaces to highlight encryption dependencies and key-rotation risks.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Custom Jobs can be configured to run on a private VPC network (VPC-SC or VPC-hosted training). In that case the job is associated with the chosen `gcp-compute-network`, allowing Overmind to show ingress/egress paths and potential network exposure.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Vertex AI executes the workload under a user-specified or default service account. The job’s permissions—and hence its ability to read data, write artefacts or call other Google APIs—are determined by this `gcp-iam-service-account`. Overmind links them to flag overly-privileged identities.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Training data, intermediate checkpoints and exported models are commonly read from or written to Cloud Storage. The Custom Job specifies bucket URIs (e.g., `gs://my-dataset/*`, `gs://my-model-output/`). Overmind connects the job to each referenced `gcp-storage-bucket` so you can assess data residency and access controls.
