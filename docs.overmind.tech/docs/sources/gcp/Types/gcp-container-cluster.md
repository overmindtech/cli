---
title: GCP Container Cluster
sidebar_label: gcp-container-cluster
---

Google Kubernetes Engine (GKE) Container Clusters provide managed Kubernetes control-planes and node infrastructure on Google Cloud Platform. A cluster groups together one or more node pools running containerised workloads, and exposes both the Kubernetes API server and optional add-ons such as Cloud Monitoring, Cloud Logging, Workload Identity and Binary Authorisation.  
For a full description of the service see the official Google documentation: https://cloud.google.com/kubernetes-engine/docs

**Terrafrom Mappings:**

- `google_container_cluster.id`

## Supported Methods

- `GET`: Get a gcp-container-cluster by its "locations|clusters"
- ~~`LIST`~~
- `SEARCH`: Search for GKE clusters in a location. Use the format "location" or the full resource name supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A cluster can be configured to encrypt Kubernetes secrets and etcd data at rest using a customer-managed Cloud KMS crypto key. When customer-managed encryption is enabled, the cluster stores the resource ID of the key that protects its control-plane data, creating a link between the cluster and the KMS crypto key.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every GKE cluster is deployed into a VPC network. All control-plane and node traffic flows inside this network, and the cluster stores the name of the network it belongs to, creating a relationship with the corresponding gcp-compute-network resource.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If a node pool is configured to run on sole-tenant nodes, GKE provisions or attaches to Compute Engine node groups for placement. The cluster will therefore reference any node groups used by its node pools.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within the chosen VPC, a cluster is attached to one or more subnetworks to allocate IP ranges for nodes, pods and services. The subnetwork resource(s) appear in the cluster’s configuration and are linked to the cluster.

### [`gcp-container-node-pool`](/sources/gcp/Types/gcp-container-node-pool)

A cluster is composed of one or more node pools that provide the actual worker nodes. Each node pool references its parent cluster, and the cluster maintains a list of all associated node pools.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

GKE uses service accounts for both the control-plane (Google-managed) and the nodes (user-specified or default). Additionally, Workload Identity maps Kubernetes service accounts to IAM service accounts. Any service account configured for node pools, Workload Identity or authorised networks will be linked to the cluster.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Audit logs and event streams originating from a GKE cluster can be exported via Logging sinks to Pub/Sub topics for downstream processing. When such a sink targets a Pub/Sub topic, the cluster indirectly references that topic, creating a link captured by Overmind.
