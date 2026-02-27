---
title: GCP Ai Platform Batch Prediction Job
sidebar_label: gcp-ai-platform-batch-prediction-job
---

A **Batch Prediction Job** in Google Cloud’s AI Platform (now part of Vertex AI) lets you run large-scale, asynchronous inference on a saved Machine Learning model. Instead of serving predictions request-by-request, you supply a dataset stored in Cloud Storage or BigQuery and the service spins up the necessary compute, distributes the workload, writes the predictions to your chosen destination, and then shuts itself down. This is ideal for one-off or periodic scoring of very large datasets.  
Official documentation: https://cloud.google.com/vertex-ai/docs/predictions/batch-predictions

## Supported Methods

* `GET`: Get a gcp-ai-platform-batch-prediction-job by its "locations|batchPredictionJobs"
* ~~`LIST`~~
* `SEARCH`: Search Batch Prediction Jobs within a location. Use the location name e.g., 'us-central1'

## Possible Links

### [`gcp-ai-platform-endpoint`](/sources/gcp/Types/gcp-ai-platform-endpoint)

A Batch Prediction Job can read from a Model that is already deployed to an Endpoint; when that is the case the job records the Endpoint name it referenced, creating this link.

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

Every Batch Prediction Job must specify the Model it will use for inference. The job stores the fully-qualified model resource name, creating a direct dependency on this Model.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

The job may take its input instances from a BigQuery table or write its prediction outputs to one. When either the source or destination is a BigQuery table, that table is linked to the job.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption keys (CMEK) are chosen, the Batch Prediction Job references the CryptoKey that encrypts the job metadata and any intermediate files, producing this link.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When the job is configured for private service access, it is attached to a specific VPC network for egress. That VPC network is therefore related to, and linked from, the job.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The Batch Prediction Job executes under a user-specified or default service account, which needs permission to read the model and the input data and to write outputs. That execution identity is linked here.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets are commonly used both for the input artefacts (CSV/JSON/TFRecord files) and for the output prediction files. Any bucket mentioned in the job’s specification is linked to the job.
