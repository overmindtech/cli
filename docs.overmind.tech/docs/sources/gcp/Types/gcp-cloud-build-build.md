---
title: GCP Cloud Build Build
sidebar_label: gcp-cloud-build-build
---

A GCP Cloud Build Build represents a single execution of Google Cloud Build, Google’s fully-managed continuous integration and delivery service. A build encapsulates the series of build steps, source code location, build artefacts, substitutions and metadata that are executed within an isolated builder environment. Each build is uniquely identified by its `name` (formatted as `projects/{projectId}/builds/{buildId}`) and records status, timing information, logs location and any images or other artefacts produced.  
For full details see the official documentation: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds

## Supported Methods

- `GET`: Get a gcp-cloud-build-build by its "name"
- `LIST`: List all gcp-cloud-build-build
- ~~`SEARCH`~~

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

If the build definition contains a step that builds and pushes a Docker image, the resulting image is usually pushed to Artifact Registry. The build therefore produces — and is linked to — one or more `gcp-artifact-registry-docker-image` resources representing the images it published.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every Cloud Build execution runs under a specific IAM service account (commonly the project-level Cloud Build service account or a custom account) which grants it permissions to fetch source, write logs and push artefacts. The build is thus associated with the `gcp-iam-service-account` used during its execution.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

Cloud Build streams build logs to Cloud Logging; organisations often route these logs into dedicated Logging buckets for retention or analysis. When such routing is configured, the build’s log entries will appear in (and therefore relate to) the relevant `gcp-logging-bucket`.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Source code for a build can be fetched from a Cloud Storage bucket, and build logs or artefact archives can also be stored in buckets created by Cloud Build (e.g. `gs://{projectId}_cloudbuild`). Consequently, a build may read from or write to one or more `gcp-storage-bucket` resources.
