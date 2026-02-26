---
title: GCP Container Cluster
sidebar_label: gcp-container-cluster
---

Google Kubernetes Engine (GKE) Container Clusters provide fully-managed Kubernetes control planes running on Google Cloud. A cluster groups the Kubernetes control plane and the worker nodes that run your containerised workloads, and exposes a single API endpoint for deployment and management. Clusters can be regional or zonal, support autoscaling, automatic upgrades and many advanced networking, security and observability features.  
Official documentation: https://cloud.google.com/kubernetes-engine/docs/concepts/kubernetes-engine-overview

**Terrafrom Mappings:**

  * `google_container_cluster.id`

## Supported Methods

* `GET`: Get a gcp-container-cluster by its "locations|clusters"
* ~~`LIST`~~
* `SEARCH`: Search for GKE clusters in a location. Use the format "location" or the full resource name supported for terraform mappings.

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

GKE can export usage metering and cost allocation data, as well as logs via Cloud Logging sinks, to a BigQuery dataset. When a cluster is configured for resource usage metering, it is linked to the destination dataset.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Clusters may use a customer-managed encryption key (CMEK) from Cloud KMS to encrypt Kubernetes Secrets and other etcd data at rest. The CMEK key configured for a cluster or for its persistent disks is therefore related.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

A specific key version is referenced by the cluster for CMEK encryption. Rotating the key version affects the cluster’s data-at-rest encryption.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every cluster is deployed into a VPC network; all control-plane and node traffic flows across this network. The network selected during cluster creation is linked here.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If the cluster uses sole-tenant nodes or node auto-provisioning, the underlying Compute Engine Node Groups that host GKE nodes are related to the cluster.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Clusters (and their node pools) are placed in one or more subnets within the VPC for pod and service IP ranges. These subnetworks are therefore linked to the cluster.

### [`gcp-container-node-pool`](/sources/gcp/Types/gcp-container-node-pool)

A cluster contains one or more node pools that define the configuration of its worker nodes (machine type, autoscaling settings, etc.). Each node pool resource is directly associated with its parent cluster.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

GKE uses IAM service accounts for the control plane, node VMs and workload identity. Service accounts granted to the cluster (e.g., Google APIs service agent, node service account) are linked.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Cluster audit logs, events or notifications can be exported to a Pub/Sub topic (e.g., via Log Sinks or Notification Channels). Any topic configured as a destination for the cluster is related here.