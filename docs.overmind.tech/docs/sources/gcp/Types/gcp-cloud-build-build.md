---
title: GCP Cloud Build Build
sidebar_label: gcp-cloud-build-build
---

A **Cloud Build Build** represents a single execution of Google Cloud Build, Google Cloud’s CI/CD service. Each build contains one or more build steps (Docker containers) that run in sequence or in parallel to compile code, run tests, or package and deploy artefacts. Metadata recorded on the build includes its source, substitutions, images, logs, secrets used, time-stamps, and overall status.  
See the official documentation for full details: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds

## Supported Methods

* `GET`: Get a gcp-cloud-build-build by its "name"
* `LIST`: List all gcp-cloud-build-build
* ~~`SEARCH`~~

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

A build often produces container images and pushes them to Artifact Registry. Overmind links the build to every `gcp-artifact-registry-docker-image` whose digest or tag is declared in the build’s `images` field.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Builds can be configured to decrypt secrets with Cloud KMS. If the build specification references a KMS key (for example in `secretEnv`), Overmind records a link to the corresponding `gcp-cloud-kms-crypto-key`.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Cloud Build runs under a service account (`serviceAccount` field). The build is therefore linked to the `gcp-iam-service-account` that actually executes the build steps and accesses other resources.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

Build logs are written to Cloud Logging and can be routed into a custom logging bucket. If log sink routing points the build’s logs to a specific `gcp-logging-bucket`, Overmind associates the two objects.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

Secrets injected into build steps via `secretEnv` or `availableSecrets` are stored in Secret Manager. A link is created between the build and every `gcp-secret-manager-secret` it consumes.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Build can pull its source from a Cloud Storage bucket and write build logs or artefacts back to buckets (e.g. via the `logsBucket` or `artifacts` fields). These buckets appear as related `gcp-storage-bucket` resources.
