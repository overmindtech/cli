---
title: GCP Dataproc Cluster
sidebar_label: gcp-dataproc-cluster
---

A Google Cloud Dataproc Cluster is a managed group of Compute Engine virtual machines configured to run big-data workloads such as Apache Hadoop, Spark, Hive and Presto. Dataproc abstracts away the operational overhead of provisioning, configuring and scaling the underlying infrastructure, allowing you to launch fully-featured clusters in minutes and shut them down just as quickly. See the official documentation for full details: https://cloud.google.com/dataproc/docs/concepts/overview

**Terrafrom Mappings:**

* `google_dataproc_cluster.name`

## Supported Methods

* `GET`: Get a gcp-dataproc-cluster by its "name"
* `LIST`: List all gcp-dataproc-cluster
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption keys (CMEK) are enabled, a Dataproc Cluster references a Cloud KMS Crypto Key to encrypt the persistent disks attached to its virtual machines.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

Each node in a Dataproc Cluster boots from a specific Compute Engine image (e.g., a Dataproc-prebuilt image or a custom image), so the cluster has a dependency on that image.

### [`gcp-compute-instance-group-manager`](/sources/gcp/Types/gcp-compute-instance-group-manager)

Dataproc automatically creates Managed Instance Groups (MIGs) for the primary, worker and optional secondary-worker node pools; these MIGs are children of the Dataproc Cluster.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

The cluster’s VMs are attached to a particular VPC network, dictating their reachability, firewall rules and routing behaviour.

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

If the cluster is deployed on sole-tenant nodes, it is associated with a Compute Node Group that provides dedicated hardware isolation.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Within the selected VPC, the Dataproc Cluster attaches its instances to a specific subnetwork where IP addressing, Private Google Access and regional placement are defined.

### [`gcp-container-cluster`](/sources/gcp/Types/gcp-container-cluster)

For Dataproc on GKE deployments, the Dataproc Cluster is layered on top of an existing Google Kubernetes Engine cluster, creating a parent–child relationship.

### [`gcp-container-node-pool`](/sources/gcp/Types/gcp-container-node-pool)

When running Dataproc on GKE, the workloads execute on one or more GKE node pools; the Dataproc service references these node pools for capacity.

### [`gcp-dataproc-autoscaling-policy`](/sources/gcp/Types/gcp-dataproc-autoscaling-policy)

A Dataproc Cluster can be bound to an Autoscaling Policy that dynamically adjusts the number of worker nodes based on workload metrics.

### [`gcp-dataproc-cluster`](/sources/gcp/Types/gcp-dataproc-cluster)

Clusters can reference other clusters as templates or in workflows that orchestrate multiple clusters; Overmind represents these peer or predecessor relationships with a self-link.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The VMs within the cluster run under one or more IAM Service Accounts that grant them permissions to access other Google Cloud services.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

During creation, the cluster specifies Cloud Storage buckets for staging, temp and log output, making those buckets upstream dependencies.
