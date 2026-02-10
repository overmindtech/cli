---
title: GCP Container Node Pool
sidebar_label: gcp-container-node-pool
---

A Google Cloud Platform (GCP) Container Node Pool is a logical grouping of worker nodes within a Google Kubernetes Engine (GKE) cluster. All nodes in a pool share the same configuration (machine type, disk size, metadata, labels, etc.) and are managed as a single unit for operations such as upgrades, autoscaling and maintenance. Node pools allow you to mix and match node types inside a single cluster, enabling workload-specific optimisation, cost control and security hardening.  
Official documentation: https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools

**Terrafrom Mappings:**

- `google_container_node_pool.id`

## Supported Methods

- `GET`: Get a gcp-container-node-pool by its "locations|clusters|nodePools"
- ~~`LIST`~~
- `SEARCH`: Search GKE Node Pools within a cluster. Use "[location]|[cluster]" or the full resource name supported by Terraform mappings: "[project]/[location]/[cluster]/[node_pool_name]"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A node pool can be configured to use a Cloud KMS CryptoKey for at-rest encryption of node boot disks or customer-managed encryption keys (CMEK) for GKE secrets. Overmind links the node pool to the KMS key that protects its data, allowing you to trace encryption dependencies.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

When a node pool is created on sole-tenant nodes, GKE provisions the underlying Compute Engine Node Group that hosts those VMs. Linking highlights which Node Group provides the physical tenancy for the pool’s nodes.

### [`gcp-container-cluster`](/sources/gcp/Types/gcp-container-cluster)

Every node pool belongs to exactly one GKE cluster. This parent-child relationship is surfaced so you can quickly navigate from a pool to its cluster and understand cluster-level configuration and risk.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each VM in a node pool runs as an IAM service account (often the “default” compute service account or a custom node service account). Overmind links the pool to that service account to expose permissions granted to workloads running on the nodes.
