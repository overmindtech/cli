---
title: GCP Run Service
sidebar_label: gcp-run-service
---

Google Cloud Run Service is a fully-managed compute platform that automatically scales stateless containers on demand. A Service represents the user-facing abstraction of your application, managing one or more immutable Revisions of a container image and routing traffic to them. It provides configuration for networking, environment variables, secrets, concurrency, autoscaling and identity.  
Official documentation: https://cloud.google.com/run/docs

**Terrafrom Mappings:**

  * `google_cloud_run_v2_service.id`

## Supported Methods

* `GET`: Get a gcp-run-service by its "locations|services"
* ~~`LIST`~~
* `SEARCH`: Search for gcp-run-service by its "locations"

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

A Cloud Run Service deploys one specific container image; most commonly this image is stored in Artifact Registry. The link shows which image version the Service’s active Revision is based on.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the Service uses customer-managed encryption keys (CMEK) for at-rest encryption of logs, volumes or secrets, it will reference a Cloud KMS Crypto Key.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When the Service is configured with a VPC connector for egress or to reach private resources, it ultimately attaches to a specific Compute Network.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

The VPC connector also targets a concrete Subnetwork; this link identifies the precise subnet through which the Service’s traffic is routed.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

A Cloud Run Service runs with a dedicated runtime identity. This Service Account is used for accessing other Google Cloud resources and defines the permissions available to the container.

### [`gcp-run-revision`](/sources/gcp/Types/gcp-run-revision)

Each update to configuration or container image creates a new Revision. The Service points traffic to one or more of these Revisions; the link maps the parent-child relationship.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

Environment variables or mounted volumes in the Service can be sourced from Secret Manager. Linked secrets indicate which sensitive values are injected at runtime.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

If Cloud SQL connections are configured via the Cloud SQL Auth Proxy side-car or built-in integration, the Service will reference one or more Cloud SQL instances.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

The Service may access files in Cloud Storage for static assets or as mounted volumes (Cloud Run volumes). Buckets listed here are those explicitly referenced by environment variables, IAM permissions or volume mounts.