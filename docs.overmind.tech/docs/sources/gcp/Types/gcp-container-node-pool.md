---
title: GCP Container Node Pool
sidebar_label: gcp-container-node-pool
---

Google Kubernetes Engine (GKE) runs worker nodes in groups called _node pools_.  
Each pool defines the machine type, disk configuration, Kubernetes version and other attributes for the virtual machines that will back your workloads, and can be scaled or upgraded independently from the rest of the cluster.  
Official documentation: https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools

**Terrafrom Mappings:**

- `google_container_node_pool.id`

## Supported Methods

- `GET`: Get a gcp-container-node-pool by its "locations|clusters|nodePools"
- ~~`LIST`~~
- `SEARCH`: Search GKE Node Pools within a cluster. Use "[location]|[cluster]" or the full resource name supported by Terraform mappings: "[project]/[location]/[cluster]/[node_pool_name]"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

When customer-managed encryption keys (CMEK) are enabled for node disks, the node pool stores a reference to the Cloud KMS crypto key that encrypts each node’s boot and attached data volumes.

### [`gcp-compute-instance-group-manager`](/sources/gcp/Types/gcp-compute-instance-group-manager)

Every node pool is implemented as a regional or zonal Managed Instance Group (MIG) that GKE creates and controls; the Instance Group Manager handles the lifecycle of the virtual machines that make up the pool.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Nodes launched by the pool are attached to a specific VPC network (and its associated routes and firewall rules), so the pool maintains a link to the Compute Network used by the cluster.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If the node pool is configured to run on sole-tenant nodes, it will reference the Compute Node Group that represents the underlying dedicated hosts reserved for those nodes.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

The pool records the particular subnetwork into which its nodes are placed, controlling the IP range from which node addresses are allocated.

### [`gcp-container-cluster`](/sources/gcp/Types/gcp-container-cluster)

A node pool is a child resource of a GKE cluster; this link identifies the parent `gcp-container-cluster` that owns and orchestrates the pool.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Each node runs with a Google service account that provides credentials for pulling container images, writing logs, and calling Google APIs. The pool stores a reference to that IAM Service Account.
