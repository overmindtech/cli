---
title: GCP Run Revision
sidebar_label: gcp-run-revision
---

A Cloud Run Revision represents an immutable snapshot of the code and configuration that Cloud Run executes. Every time you deploy a new container image or change the runtime configuration of a Cloud Run Service, a new Revision is created and given a unique name. The Revision stores details such as the container image reference, environment variables, scaling limits, traffic settings, networking options and the service account under which the workload runs. Official documentation: https://docs.cloud.google.com/run/docs/managing/revisions

## Supported Methods

- `GET`: Get a gcp-run-revision by its "locations|services|revisions"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-run-revision by its "locations|services"

## Possible Links

### [`gcp-artifact-registry-docker-image`](/sources/gcp/Types/gcp-artifact-registry-docker-image)

The Revision’s `container.image` field points to a Docker image that is normally stored in Artifact Registry (or the older Container Registry). Overmind therefore links the Revision to the exact image digest it deploys, so you can see what code is really running.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the Revision mounts secrets or other resources that are encrypted with Cloud KMS, those crypto-keys are surfaced as links. This helps you understand which keys would be required to decrypt data at runtime.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When a Revision is configured with a Serverless VPC Connector or egress settings that reference a particular VPC network, the corresponding `compute.network` is linked. This reveals the network perimeter through which outbound traffic may flow.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Similarly, a Revision may target a specific sub-network (for example `vpcAccess.connectorSubnetwork`). Overmind links the Revision to that `compute.subnetwork` so you can trace which CIDR ranges and routes apply.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each Revision runs with an IAM service account specified in its `serviceAccountName` field. Linking to the service account lets you inspect the permissions that the workload inherits.

### [`gcp-run-service`](/sources/gcp/Types/gcp-run-service)

A Revision belongs to exactly one Cloud Run Service. The link to the parent Service shows the traffic allocation, routing configuration and other higher-level settings that govern how the Revision is invoked.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

If the Revision’s metadata includes Cloud SQL connection strings (via the `cloudSqlInstances` setting), Overmind links to the referenced Cloud SQL instances, making database dependencies explicit.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Revisions can mount Cloud Storage buckets using Cloud Storage FUSE volumes or reference buckets through environment variables. When such configuration is detected, the corresponding buckets are linked so you can assess data-at-rest exposure.
