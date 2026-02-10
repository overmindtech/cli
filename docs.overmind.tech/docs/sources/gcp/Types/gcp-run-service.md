---
title: GCP Run Service
sidebar_label: gcp-run-service
---

Cloud Run Service is a fully-managed container execution environment that lets you run stateless HTTP containers on demand within Google Cloud. A Service represents the top-level Cloud Run resource, providing a stable URL, traffic splitting, configuration, and revision management for your containerised workload. For full details see the Google Cloud documentation: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services

**Terrafrom Mappings:**

- `google_cloud_run_v2_service.id`

## Supported Methods

- `GET`: Get a gcp-run-service by its "locations|services"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-run-service by its "locations"

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

A Cloud Run Service pulls its container image from Artifact Registry (or Container Registry). The linked `gcp-artifact-registry-docker-image` represents the specific image digest or tag referenced in the Service spec.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the Service’s container image or any attached Secret Manager secret is encrypted with a customer-managed encryption key (CMEK), the Cloud Run Service will be linked to the corresponding `gcp-cloud-kms-crypto-key`.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When a Cloud Run Service is configured with a Serverless VPC Access connector, it attaches to a VPC network to reach private resources. That network is represented here as a `gcp-compute-network`.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

The Serverless VPC Access connector also lives on a particular subnetwork. The Cloud Run Service therefore relates to the `gcp-compute-subnetwork` used for outbound traffic.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every Cloud Run Service executes with an identity (the “service account” set in the Service’s `executionEnvironment` or `serviceAccount`). This runtime identity is captured as a link to `gcp-iam-service-account`.

### [`gcp-run-revision`](/sources/gcp/Types/gcp-run-revision)

Each deployment of a Cloud Run Service creates an immutable Revision. The Service maintains traffic routing rules among its Revisions, so it links to one or more `gcp-run-revision` resources.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

Environment variables or mounted volumes can reference secrets stored in Secret Manager. Any such secret referenced by the Service or its Revisions appears as a `gcp-secret-manager-secret` link.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

If the Service includes a Cloud SQL connection string (via the `cloudsql-instances` annotation), Overmind records a relationship to the corresponding `gcp-sql-admin-instance`.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Run Services may interact with Cloud Storage—for example, by having a URL environment variable or event trigger configuration. Where such a bucket name is detected in the Service configuration, it is linked here as `gcp-storage-bucket`.
