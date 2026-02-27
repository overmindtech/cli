---
title: GCP Run Revision
sidebar_label: gcp-run-revision
---

A Cloud Run **Revision** is an immutable snapshot of a Cloud Run Service configuration at a particular point in time. Each time you deploy new code or change configuration, Cloud Run automatically creates a new revision and routes traffic according to your settings. A revision defines the container image to run, environment variables, resource limits, networking options, service account, secret mounts and more. Once created, a revision can never be modified – you can only create a new one.  
Official documentation: https://cloud.google.com/run/docs/reference/rest/v1/namespaces.revisions

## Supported Methods

* `GET`: Get a gcp-run-revision by its "locations|services|revisions"
* ~~`LIST`~~
* `SEARCH`: Search for gcp-run-revision by its "locations|services"

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

The container image specified in the revision is often stored in Artifact Registry. The revision therefore has a **uses-image** relationship with the referenced Docker image.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the revision is configured with a customer-managed encryption key (CMEK) for encrypted secrets or volumes, it will reference the corresponding Cloud KMS Crypto Key.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When a revision is set up to use Serverless VPC Access, it connects to a specific VPC network, creating a **connects-to-network** relationship.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

The Serverless VPC Access connector used by the revision is attached to a particular subnetwork, so the revision is indirectly linked to that subnetwork.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each revision runs with an IAM service account whose permissions govern outbound calls and resource access. The revision therefore **runs-as** the referenced service account.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

The revision is a child resource of a Cloud Run Service. All traffic routing and lifecycle events are managed at the service level.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

Environment variables or mounted volumes in the revision can pull values from Secret Manager. This establishes a **consumes-secret** relationship.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

If the revision defines Cloud SQL connections, it will list one or more Cloud SQL instances it can connect to through the Cloud SQL proxy.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

A revision may read from or write to Cloud Storage buckets (for example for static assets or generated files) when granted the appropriate IAM permissions, creating a potential dependency on those buckets.
